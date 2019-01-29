pragma solidity ^0.5.0;

/// @title Library implementing an array type which allows O(1) lookups on values.
/// @author Piper Merriam <pipermerriam@gmail.com>, Eric Olszewski <eolszewski@gmail.com>
/// Adapted from https://github.com/ethpm/ethereum-indexed-enumerable-set-lib/blob/master/contracts/IndexedEnumerableSetLib.sol
library StringSetLib
{
    struct StringSet {
        string[] values;
        mapping(bytes32 => bool) exists;
        mapping(bytes32 => uint) indices;
    }

    modifier inBounds(StringSet storage self, uint index) {
        require(index < self.values.length);
        _;
    }

    modifier notEmpty(StringSet storage self) {
        require(self.values.length != 0);
        _;
    }

    function hash(string memory s)
        private
        pure
        returns (bytes32)
    {
        return keccak256(abi.encodePacked(s));
    }

    function get(StringSet storage self, uint index)
        public
        view
        inBounds(self, index)
        returns (string memory)
    {
        return self.values[index];
    }

    function set(StringSet storage self, uint index, string memory value)
        public
        inBounds(self, index)
        returns (bool)
    {
        bytes32 hvalue = hash(value);
        if (self.exists[hvalue]) {
            return false;
        }

        self.values[index] = value;
        self.exists[hvalue] = true;
        self.indices[hvalue] = index;
        return true;
    }

    function add(StringSet storage self, string memory value)
        public
        returns (bool)
    {
        if (self.exists[hash(value)]) {
            return false;
        }

        self.indices[hash(value)] = self.values.length;
        self.values.push(value);
        self.exists[hash(value)] = true;
        return true;
    }

    function remove(StringSet storage self, string memory value)
        public
        returns (bool)
    {
        if (!self.exists[hash(value)]) {
            return false;
        }
        uint index = indexOf(self, value);
        pop(self, index);
        return true;
    }

    function pop(StringSet storage self, uint index)
        public
        inBounds(self, index)
        returns (string memory)
    {
        string memory value = get(self, index);

        if (index != self.values.length - 1) {
            string memory lastValue = last(self);
            bytes32 hlastValue = hash(lastValue);
            self.exists[hlastValue] = false;
            set(self, index, lastValue);
            self.indices[hlastValue] = index;
        }
        self.values.length -= 1;

        bytes32 hvalue = hash(value);
        delete self.indices[hvalue];
        delete self.exists[hvalue];

        return value;
    }

    function replace(StringSet storage self, string memory old, string memory nu)
        public
        returns (bool)
    {
        return remove(self, old) && add(self, nu);
    }

    function first(StringSet storage self)
        public
        view
        notEmpty(self)
        returns (string memory)
    {
        return get(self, 0);
    }

    function last(StringSet storage self)
        public
        view
        notEmpty(self)
        returns (string memory)
    {
        return get(self, self.values.length - 1);
    }

    function indexOf(StringSet storage self, string memory value)
        public
        view
        returns (uint)
    {
        bytes32 hvalue = hash(value);
        if (!self.exists[hvalue]) {
            return uint(-1);
        }
        return self.indices[hvalue];
    }

    function contains(StringSet storage self, string memory value)
        public
        view
        returns (bool)
    {
        return self.exists[hash(value)];
    }

    function size(StringSet storage self)
        public
        view
        returns (uint)
    {
        return self.values.length;
    }
}