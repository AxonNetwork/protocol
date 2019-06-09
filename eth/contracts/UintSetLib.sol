pragma solidity ^0.5.0;

/// @title Library implementing an array type which allows O(1) lookups on values.
/// @author Piper Merriam <pipermerriam@gmail.com>, Eric Olszewski <eolszewski@gmail.com>
/// Adapted from https://github.com/ethpm/ethereum-indexed-enumerable-set-lib/blob/master/contracts/IndexedEnumerableSetLib.sol
library UintSetLib {

    struct UintSet {
        uint[] values;
        mapping(uint => bool) exists;
        mapping(uint => uint) indices;
    }

    modifier inBounds(UintSet storage self, uint index) {
        require(index < self.values.length);
        _;
    }

    modifier notEmpty(UintSet storage self) {
        require(self.values.length != 0);
        _;
    }

    function get(UintSet storage self, uint index) public view
        inBounds(self, index)
        returns (uint)
    {
        return self.values[index];
    }

    function set(UintSet storage self, uint index, uint value) public
        inBounds(self, index)
        returns (bool)
    {
        if (self.exists[value])
            return false;
        self.values[index] = value;
        self.exists[value] = true;
        self.indices[value] = index;
        return true;
    }

    function add(UintSet storage self, uint value) public
        returns (bool)
    {
        if (self.exists[value])
            return false;
        self.indices[value] = self.values.length;
        self.values.push(value);
        self.exists[value] = true;
        return true;
    }

    function remove(UintSet storage self, uint value) public
        returns (bool)
    {
        if (!self.exists[value])
            return false;
        uint index = indexOf(self, value);
        pop(self, index);
        return true;
    }

    function pop(UintSet storage self, uint index) public
        inBounds(self, index)
        returns (uint)
    {
        uint value = get(self, index);

        if (index != self.values.length - 1) {
            uint lastValue = last(self);
            self.exists[lastValue] = false;
            set(self, index, lastValue);
            self.indices[lastValue] = index;
        }
        self.values.length -= 1;

        delete self.indices[value];
        delete self.exists[value];

        return value;
    }

    function first(UintSet storage self) public view
        notEmpty(self)
        returns (uint)
    {
        return get(self, 0);
    }

    function last(UintSet storage self) public view
        notEmpty(self)
        returns (uint)
    {
        return get(self, self.values.length - 1);
    }

    function indexOf(UintSet storage self, uint value) public view
        returns (uint)
    {
        if (!self.exists[value])
            return uint(-1);
        return self.indices[value];
    }

    function contains(UintSet storage self, uint value) public view
        returns (bool)
    {
        return self.exists[value];
    }

    function size(UintSet storage self) public view
        returns (uint)
    {
        return self.values.length;
    }
}