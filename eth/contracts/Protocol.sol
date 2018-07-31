pragma solidity ^0.4.24;
// pragma experimental ABIEncoderV2;

contract Protocol
{
    mapping(address => string) public usernamesByAddress;
    mapping(bytes32 => address) public addressesByUsername;

    struct Repo {
        bool exists;

        mapping(bytes32 => string) refs;
        string[] refsList;

        mapping(bytes32 => bool) pushers;
        string[] pushersList;

        bool isPrivate;
        mapping(bytes32 => bool) pullers;
        string[] pullersList;
    }

    mapping(bytes32 => Repo) repositories;

    event LogSetUsername(address addr, string username);

    constructor()
        public
    {

    }

    function setUsername(string username)
        public
    {
        require(bytes(username).length > 0);
        require(bytes(usernamesByAddress[msg.sender]).length == 0);

        bytes32 usernameHash = hashString(username);
        require(addressesByUsername[usernameHash] == 0x0);

        usernamesByAddress[msg.sender] = username;
        addressesByUsername[usernameHash] = msg.sender;

        emit LogSetUsername(msg.sender, username);
    }

    function createRepository(string repoID)
        public
    {
        require(bytes(repoID).length > 0);

        string memory username = usernamesByAddress[msg.sender];
        require(bytes(username).length > 0);

        bytes32 repoIDHash = hashString(repoID);
        bytes32 usernameHash = hashString(username);

        require(repositories[repoIDHash].exists == false);

        repositories[repoIDHash].exists = true;
        repositories[repoIDHash].pushersList.push(username);
        repositories[repoIDHash].pushers[usernameHash] = true;
    }

    function repositoryExists(string repoID)
        public
        view
        returns (bool)
    {
        require(bytes(repoID).length > 0);
        bytes32 repoIDHash = hashString(repoID);
        return repositories[repoIDHash].exists;
    }

    function userHasPullAccess(string username, string repoID)
        public
        view
        returns (bool)
    {
        bytes32 repoIDHash = hashString(repoID);
        if (repositories[repoIDHash].isPrivate == false) {
            return true;
        }

        bytes32 usernameHash = hashString(username);
        return repositories[repoIDHash].pullers[usernameHash];
    }

    function userHasPushAccess(string username, string repoID)
        public
        view
        returns (bool)
    {
        bytes32 repoIDHash = keccak256(abi.encodePacked(repoID));
        bytes32 usernameHash = keccak256(abi.encodePacked(username));
        return repositories[repoIDHash].pushers[usernameHash];
    }

    function updateRef(string repoID, string refName, string commitHash)
        public
    {
        require(userHasPushAccess(usernamesByAddress[msg.sender], repoID));

        bytes32 repoIDHash = hashString(repoID);
        Repo storage repo = repositories[repoIDHash];

        bytes32 refNameHash = hashString(refName);
        if (bytes(repo.refs[refNameHash]).length == 0) {
            repo.refsList.push(refName);
        }
        repo.refs[refNameHash] = commitHash;
    }

    function numRefs(string repoID)
        public
        view
        returns (uint)
    {
        bytes32 repoIDHash = hashString(repoID);
        return repositories[repoIDHash].refsList.length;
    }

    function getRefs(string repoID, uint page)
        public
        view
        returns (bytes)
    {
        bytes32 repoIDHash = hashString(repoID);
        Repo storage repo = repositories[repoIDHash];
        require(repo.exists);

        string refName;
        string commit;
        bytes32 refNameHash;
        uint len = 0;
        uint start = page * 10;
        for (uint i = 0; i + start < repo.refsList.length; i++) {
            refNameHash = hashString(repo.refsList[i + start]);
            refName = repo.refsList[i + start];
            commit = repo.refs[refNameHash];
            len += 32 + bytes(refName).length + 32 + bytes(commit).length;
        }

        bytes memory bs = new bytes(len);
        uint written = 0;
        for (i = 0; i + start < repo.refsList.length; i++) {
            refNameHash = hashString(repo.refsList[i + start]);
            refName = repo.refsList[i + start];
            commit = repo.refs[refNameHash];

            writeUint(bytes(refName).length, bs, written);
            written += 32;
            writeBytes(bytes(refName), bs, written);
            written += bytes(refName).length;

            writeUint(bytes(commit).length, bs, written);
            written += 32;
            writeBytes(bytes(commit), bs, written);
            written += bytes(commit).length;
        }

        return bs;
    }

    function writeUint(uint x, bytes memory bs, uint offset)
        private
        pure
    {
        bytes32 b = bytes32(x);
        for (uint i = 0; i < 32; i++) {
            bs[i + offset] = b[i];
        }
    }

    function writeBytes(bytes src, bytes dest, uint offset)
        private
        pure
    {
        for (uint i = 0; i < src.length; i++) {
            dest[i + offset] = src[i];
        }
    }

    function hashString(string s)
        private
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(s));
    }
}