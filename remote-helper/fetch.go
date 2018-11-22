package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"log"

	"github.com/pkg/errors"
	gitplumbing "gopkg.in/src-d/go-git.v4/plumbing"
)

func fetchFromCommit_packfile(commitHash string) error {
	hash, err := hex.DecodeString(commitHash)
	if err != nil {
		return err
	} else if Repo.HasObject(hash) {
		return nil
	}

	ch, err := client.FetchFromCommit(context.Background(), repoID, Repo.Path, commitHash)
	if err != nil {
		return err
	}

	packfiles := make(map[string]io.WriteCloser)
	latestPercent := 0
	for pkt := range ch {
		if pkt.Error != nil {
			return pkt.Error
		}

		packfileID := hex.EncodeToString(pkt.ObjHash)

		if _, exists := packfiles[packfileID]; !exists {
			pw, err := Repo.PackfileWriter()
			if err != nil {
				return err
			}

			packfiles[packfileID] = pw
		}

		if pkt.End {
			err = packfiles[packfileID].Close()
			if err != nil {
				return errors.WithStack(err)
			}

			delete(packfiles, packfileID)

		} else {
			n, err := packfiles[packfileID].Write(pkt.Data)
			if err != nil {
				return errors.WithStack(err)
			} else if n != len(pkt.Data) {
				return errors.New("remote helper: did not fully write packet")
			}
			percentDownloaded := 0
			if pkt.ToFetch > 0 {
				percentDownloaded = int(100 * pkt.Fetched / pkt.ToFetch)
			}
			if percentDownloaded > latestPercent {
				latestPercent = percentDownloaded
				log.Printf("Progress: %d%%\n", latestPercent)
			}
		}
	}
	return nil
}

func fetchFromCommit_object(commitHash string) error {
	hash, err := hex.DecodeString(commitHash)
	if err != nil {
		return err
	} else if Repo.HasObject(hash) {
		return nil
	}

	ch, err := client.FetchFromCommit(context.Background(), repoID, Repo.Path, commitHash)
	if err != nil {
		return err
	}

	type FileStream struct {
		file       gitplumbing.EncodedObject
		fileWriter io.WriteCloser
		written    uint64
	}

	files := make(map[string]*FileStream)

	for pkt := range ch {
		if pkt.Error != nil {
			return pkt.Error
		}

		hash := hex.EncodeToString(pkt.ObjHash)

		if _, exists := files[hash]; !exists {
			obj := Repo.Storer.NewEncodedObject()
			obj.SetType(gitplumbing.ObjectType(pkt.ObjType))

			w, err := obj.Writer()
			if err != nil {
				return err
			}

			files[hash] = &FileStream{
				file:       obj,
				fileWriter: w,
				written:    0,
			}
		}

		n, err := files[hash].fileWriter.Write(pkt.Data)
		if err != nil {
			return errors.WithStack(err)
		} else if n != len(pkt.Data) {
			return errors.New("remote helper: did not fully write packet")
		}

		files[hash].written += uint64(n)
		if files[hash].written >= pkt.ObjLen {
			err = files[hash].fileWriter.Close()
			if err != nil {
				return errors.WithStack(err)
			}

			h := files[hash].file.Hash()
			if !bytes.Equal(pkt.ObjHash, h[:]) {
				return errors.Errorf("remote helper: bad checksum for %v", hash)
			}

			_, err = Repo.Storer.SetEncodedObject(files[hash].file)
			if err != nil {
				return errors.WithStack(err)
			}

			delete(files, hash)
		}
	}
	return nil
}

// var inflightLimiter = make(chan struct{}, 5)

// func init() {
// 	for i := 0; i < 5; i++ {
// 		inflightLimiter <- struct{}{}
// 	}
// }

// func fetch(hash gitplumbing.Hash) error {
// 	wg := &sync.WaitGroup{}
// 	chErr := make(chan error)

// 	wg.Add(1)
// 	go recurseObject(hash, wg, chErr)

// 	chDone := make(chan struct{})
// 	go func() {
// 		defer close(chDone)
// 		wg.Wait()
// 	}()

// 	select {
// 	case <-chDone:
// 		return nil
// 	case err := <-chErr:
// 		return err
// 	}
// }

// func recurseObject(hash gitplumbing.Hash, wg *sync.WaitGroup, chErr chan error) {
// 	defer wg.Done()

// 	objType, err := fetchAndWriteObject(hash)
// 	if err != nil {
// 		chErr <- err
// 		return
// 	}

// 	// If the object is a tree or commit, make sure we have its children
// 	switch objType {
// 	case gitplumbing.TreeObject:
// 		tree, err := Repo.TreeObject(hash)
// 		if err != nil {
// 			chErr <- errors.WithStack(err)
// 			return
// 		}

// 		for _, entry := range tree.Entries {
// 			wg.Add(1)
// 			go recurseObject(entry.Hash, wg, chErr)
// 		}

// 	case gitplumbing.CommitObject:
// 		commit, err := Repo.CommitObject(hash)
// 		if err != nil {
// 			chErr <- errors.WithStack(err)
// 			return
// 		}

// 		if commit.NumParents() > 0 {
// 			for _, phash := range commit.ParentHashes {
// 				wg.Add(1)
// 				go recurseObject(phash, wg, chErr)
// 			}
// 		}

// 		wg.Add(1)
// 		go recurseObject(commit.TreeHash, wg, chErr)
// 	}
// }

// func fetchAndWriteObject(hash gitplumbing.Hash) (gitplumbing.ObjectType, error) {
// 	<-inflightLimiter
// 	defer func() { inflightLimiter <- struct{}{} }()

// 	obj, err := Repo.Object(gitplumbing.AnyObject, hash)
// 	// The object has already been downloaded
// 	if err == nil {
// 		return obj.Type(), nil
// 	}

// 	// Fetch an object stream from the node via RPC
// 	// @@TODO: give context a timeout and make it configurable
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()

// 	objectStream, err := client.GetObject(ctx, repoID, hash[:])
// 	if err != nil {
// 		return 0, errors.WithStack(err)
// 	}
// 	defer objectStream.Close()

// 	// Write the object to disk
// 	{
// 		newobj := Repo.Storer.NewEncodedObject() // returns a &plumbing.MemoryObject{}
// 		newobj.SetType(objectStream.Type())

// 		w, err := newobj.Writer()
// 		if err != nil {
// 			return 0, errors.WithStack(err)
// 		}

// 		copied, err := io.Copy(w, objectStream)
// 		if err != nil {
// 			return 0, errors.WithStack(err)
// 		} else if uint64(copied) != objectStream.Len() {
// 			return 0, errors.Errorf("object stream bad length (copied: %v, object length: %v)", copied, objectStream.Len())
// 		}

// 		err = w.Close()
// 		if err != nil {
// 			return 0, errors.WithStack(err)
// 		}

// 		// Check the checksum
// 		if hash != newobj.Hash() {
// 			return 0, errors.Errorf("bad checksum for piece %v", hash.String())
// 		}

// 		// Write the object to disk
// 		_, err = Repo.Storer.SetEncodedObject(newobj)
// 		if err != nil {
// 			return 0, errors.WithStack(err)
// 		}
// 	}
// 	return objectStream.Type(), nil
// }
