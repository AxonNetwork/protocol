pragma solidity 0.5.0;

import "../UintSetLib.sol";

// contract BasicAccessControl
// {
//     using UintSetLib for UintSetLib.UintSet;

//     struct Repo {
//         bool exists;
//         UintSetLib.UintSet admins;
//         UintSetLib.UintSet pushers;
//         UintSetLib.UintSet pullers;
//         bool isPublic;
//     }

//     mapping(string => Repo) repos;

//     enum UserType {
//         ADMIN, PULLER, PUSHER
//     }

//     function userCan(string verb, string repoID, uint userID) public view returns (bool) {
//         Repo storage repo = repos[repoID];

//         if (verb == "pull") {
//             if (repo.isPublic) {
//                 return true;
//             }
//             return repo.pullers.contains(userID)

//         } else if (verb == "push") {
//             return repo.pushers.contains(userID);

//         } else if (verb == "modify permissions") {
//             return repo.admins.contains(userID);
//         }
//     }

//     function userIsAdmin(string memory username, string memory repoID) public view returns (bool) {
//         return repositories[hashString(repoID)].admins.contains(username);
//     }

//     function userHasPullAccess(string memory username, string memory repoID) public view returns (bool) {
//         bytes32 repoIDHash = hashString(repoID);
//         if (repositories[repoIDHash].isPublic == true) {
//             return true;
//         }
//         return repositories[repoIDHash].pullers.contains(username);
//     }

//     function userHasPushAccess(string memory username, string memory repoID) public view returns (bool) {
//         return repositories[hashString(repoID)].pushers.contains(username);
//     }

//     function addressIsAdmin(address addr, string memory repoID) public view returns (bool) {
//         return userIsAdmin(userRegistry.usernameForAddress(addr), repoID);
//     }

//     function addressHasPullAccess(address addr, string memory repoID) public view returns (bool) {
//         return userHasPullAccess(userRegistry.usernameForAddress(addr), repoID);
//     }

//     function addressHasPushAccess(address addr, string memory repoID) public view returns (bool) {
//         return userHasPushAccess(userRegistry.usernameForAddress(addr), repoID);
//     }

//     function getUserPermissions(string memory repoID, string memory username) public view returns (bool puller, bool pusher, bool admin) {
//         Repo storage repo = repositories[hashString(repoID)];
//         require(repo.exists, "repo does not exist");

//         return (
//             repo.pullers.contains(username),
//             repo.pushers.contains(username),
//             repo.admins.contains(username)
//         );
//     }

//     function setUserPermissions(string memory repoID, string memory username, bool puller, bool pusher, bool admin) public {
//         require(addressIsAdmin(msg.sender, repoID), "you are not an admin");

//         Repo storage repo = repositories[hashString(repoID)];
//         require(repo.exists, "repo does not exist");

//         if (puller) {
//             repo.pullers.add(username);
//         } else {
//             repo.pullers.remove(username);
//         }

//         if (pusher) {
//             repo.pushers.add(username);
//         } else {
//             repo.pushers.remove(username);
//         }

//         if (admin) {
//             repo.admins.add(username);
//         } else {
//             repo.admins.remove(username);
//         }
//     }
// }