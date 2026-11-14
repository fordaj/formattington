// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract Widget {
    function render(uint256 value) public pure returns (uint256) {
        return value * 2;
    }
}
