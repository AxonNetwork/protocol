// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package ethcontracts

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// ProtocolABI is the input ABI used to generate the binding from.
const ProtocolABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"usernamesByAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"refName\",\"type\":\"string\"},{\"name\":\"commitHash\",\"type\":\"string\"}],\"name\":\"updateRef\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPushAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"addressesByUsername\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"page\",\"type\":\"uint256\"}],\"name\":\"getRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"}],\"name\":\"setUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPullAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"createRepository\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"numRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"username\",\"type\":\"string\"}],\"name\":\"LogSetUsername\",\"type\":\"event\"}]"

// ProtocolBin is the compiled bytecode used for deploying new contracts.
const ProtocolBin = `0x608060405234801561001057600080fd5b50611263806100206000396000f3006080604052600436106100985763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166342dfb6da811461009d578063605a5dd8146101405780637bbaf15914610217578063a58e325c146102c2578063b84da29914610303578063ed59313a1461035e578063ede07dfe146103b7578063ee94cf6d1461044e578063f2ebfa10146104a7575b600080fd5b3480156100a957600080fd5b506100cb73ffffffffffffffffffffffffffffffffffffffff60043516610512565b6040805160208082528351818301528351919283929083019185019080838360005b838110156101055781810151838201526020016100ed565b50505050905090810190601f1680156101325780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561014c57600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261021594369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506105ac9650505050505050565b005b34801561022357600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102ae94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375094975061070a9650505050505050565b604080519115158252519081900360200190f35b3480156102ce57600080fd5b506102da6004356108b9565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b34801561030f57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526100cb94369492936024939284019190819084018382808284375094975050933594506108e19350505050565b34801561036a57600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610215943694929360249392840191908190840183828082843750949750610c879650505050505050565b3480156103c357600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102ae94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a999881019791965091820194509250829150840183828082843750949750610df69650505050505050565b34801561045a57600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610215943694929360249392840191908190840183828082843750949750610e5e9650505050505050565b3480156104b357600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610500943694929360249392840191908190840183828082843750949750610fc29650505050505050565b60408051918252519081900360200190f35b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152929091908301828280156105a45780601f10610579576101008083540402835291602001916105a4565b820191906000526020600020905b81548152906001019060200180831161058757829003601f168201915b505050505081565b336000908152602081815260408083208054825160026001831615610100026000190190921691909104601f81018590048502820185019093528281528493849361064e93918301828280156106435780601f1061061857610100808354040283529160200191610643565b820191906000526020600020905b81548152906001019060200180831161062657829003601f168201915b50505050508761070a565b151561065957600080fd5b61066286610fe8565b6000818152600260205260409020909350915061067e85610fe8565b6000818152600180850160205260409091205491925060026101009183161591909102600019019091160415156106e057600282018054600181018083556000928352602092839020885191936106dd9391909101919089019061119c565b50505b6000818152600183016020908152604090912085516107019287019061119c565b50505050505050565b6000806000836040516020018082805190602001908083835b602083106107425780518252601f199092019160209182019101610723565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106107a55780518252601f199092019160209182019101610786565b51815160209384036101000a60001901801990921691161790526040519190930181900381208a519097508a955090830193508392850191508083835b602083106108015780518252601f1990920191602091820191016107e2565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106108645780518252601f199092019160209182019101610845565b51815160209384036101000a600019018019909216911617905260408051929094018290039091206000888152600283528481208282526003019092529290205460ff1696509093505050505b505092915050565b60016020526000908152604090205473ffffffffffffffffffffffffffffffffffffffff1681565b6060600080600080600080600080606060006108fc8d610fe8565b60008181526002602052604090208054919b50995060ff16151561091f57600080fd5b600094508b600a029350600092505b60028901548385011015610a6e576109ea8960020185850181548110151561095257fe5b600091825260209182902001805460408051601f60026000196101006001871615020190941693909304928301859004850281018501909152818152928301828280156109e05780601f106109b5576101008083540402835291602001916109e0565b820191906000526020600020905b8154815290600101906020018083116109c357829003601f168201915b5050505050610fe8565b9550886002018484018154811015156109ff57fe5b9060005260206000200197508860010160008760001916600019168152602001908152602001600020965086805460018160011615610100020316600290049050888054600181600116156101000203166002900490506020016020010185019450828060010193505061092e565b846040519080825280601f01601f191660200182016040528015610a9c578160200160208202803883390190505b50915060009050600092505b60028901548385011015610c7757610acc8960020185850181548110151561095257fe5b955088600201848401815481101515610ae157fe5b600091825260208083208984526001808e019092526040909320929091018054909a50919850610b25916002610100928216159290920260001901160483836110b2565b875460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292820192610bbf9290918b91830182828015610bb35780601f10610b8857610100808354040283529160200191610bb3565b820191906000526020600020905b815481529060010190602001808311610b9657829003601f168201915b50505050508383611121565b875487546002600019610100600180861615820283019095168390049590950194610bf49484161502019091160483836110b2565b865460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292820192610c579290918a91830182828015610bb35780601f10610b8857610100808354040283529160200191610bb3565b865460019384019360029082161561010002600019019091160401610aa8565b509b9a5050505050505050505050565b6000808251111515610c9857600080fd5b336000908152602081905260409020546002600019610100600184161502019091160415610cc557600080fd5b610cce82610fe8565b60008181526001602052604090205490915073ffffffffffffffffffffffffffffffffffffffff1615610d0057600080fd5b336000908152602081815260409091208351610d1e9285019061119c565b506000818152600160209081526040808320805473ffffffffffffffffffffffffffffffffffffffff191633908117909155815181815280840183815287519382019390935286517faffa6dd92f7ba89dd7b4fdd8809b8e8d38b6431d8f41674fae86cfa06fc66d9995929488949293606085019291860191908190849084905b83811015610db7578181015183820152602001610d9f565b50505050905090810190601f168015610de45780820380516001836020036101000a031916815260200191505b50935050505060405180910390a15050565b6000806000610e0484610fe8565b60008181526002602052604090206005015490925060ff161515610e2b57600192506108b1565b610e3485610fe8565b6000928352600260209081526040808520928552600690920190529091205460ff16949350505050565b606060008060008451111515610e7357600080fd5b336000908152602081815260409182902080548351601f600260001961010060018616150201909316929092049182018490048402810184019094528084529091830182828015610f055780601f10610eda57610100808354040283529160200191610f05565b820191906000526020600020905b815481529060010190602001808311610ee857829003601f168201915b5050505050925060008351111515610f1c57600080fd5b610f2584610fe8565b9150610f3083610fe8565b60008381526002602052604090205490915060ff1615610f4f57600080fd5b60008281526002602090815260408220805460ff19166001908117825560049091018054918201808255908452928290208651610f949391909201919087019061119c565b50506000918252600260209081526040808420928452600390920190529020805460ff191660011790555050565b600080610fce83610fe8565b600090815260026020819052604090912001549392505050565b6000816040516020018082805190602001908083835b6020831061101d5780518252601f199092019160209182019101610ffe565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106110805780518252601f199092019160209182019101611061565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b8260005b602081101561111a578181602081106110cb57fe5b1a60f860020a02848483018151811015156110e257fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506001016110b6565b5050505050565b60005b835181101561119657838181518110151561113b57fe5b90602001015160f860020a900460f860020a028383830181518110151561115e57fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600101611124565b50505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106111dd57805160ff191683800117855561120a565b8280016001018555821561120a579182015b8281111561120a5782518255916020019190600101906111ef565b5061121692915061121a565b5090565b61123491905b808211156112165760008155600101611220565b905600a165627a7a72305820acd0121eabef4c6e4112431b8468071d86bf3231e3460f129d3dc653045b04ab0029`

// DeployProtocol deploys a new Ethereum contract, binding an instance of Protocol to it.
func DeployProtocol(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Protocol, error) {
	parsed, err := abi.JSON(strings.NewReader(ProtocolABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(ProtocolBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Protocol{ProtocolCaller: ProtocolCaller{contract: contract}, ProtocolTransactor: ProtocolTransactor{contract: contract}, ProtocolFilterer: ProtocolFilterer{contract: contract}}, nil
}

// Protocol is an auto generated Go binding around an Ethereum contract.
type Protocol struct {
	ProtocolCaller     // Read-only binding to the contract
	ProtocolTransactor // Write-only binding to the contract
	ProtocolFilterer   // Log filterer for contract events
}

// ProtocolCaller is an auto generated read-only Go binding around an Ethereum contract.
type ProtocolCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProtocolTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ProtocolTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProtocolFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ProtocolFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ProtocolSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ProtocolSession struct {
	Contract     *Protocol         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ProtocolCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ProtocolCallerSession struct {
	Contract *ProtocolCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ProtocolTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ProtocolTransactorSession struct {
	Contract     *ProtocolTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ProtocolRaw is an auto generated low-level Go binding around an Ethereum contract.
type ProtocolRaw struct {
	Contract *Protocol // Generic contract binding to access the raw methods on
}

// ProtocolCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ProtocolCallerRaw struct {
	Contract *ProtocolCaller // Generic read-only contract binding to access the raw methods on
}

// ProtocolTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ProtocolTransactorRaw struct {
	Contract *ProtocolTransactor // Generic write-only contract binding to access the raw methods on
}

// NewProtocol creates a new instance of Protocol, bound to a specific deployed contract.
func NewProtocol(address common.Address, backend bind.ContractBackend) (*Protocol, error) {
	contract, err := bindProtocol(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Protocol{ProtocolCaller: ProtocolCaller{contract: contract}, ProtocolTransactor: ProtocolTransactor{contract: contract}, ProtocolFilterer: ProtocolFilterer{contract: contract}}, nil
}

// NewProtocolCaller creates a new read-only instance of Protocol, bound to a specific deployed contract.
func NewProtocolCaller(address common.Address, caller bind.ContractCaller) (*ProtocolCaller, error) {
	contract, err := bindProtocol(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ProtocolCaller{contract: contract}, nil
}

// NewProtocolTransactor creates a new write-only instance of Protocol, bound to a specific deployed contract.
func NewProtocolTransactor(address common.Address, transactor bind.ContractTransactor) (*ProtocolTransactor, error) {
	contract, err := bindProtocol(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ProtocolTransactor{contract: contract}, nil
}

// NewProtocolFilterer creates a new log filterer instance of Protocol, bound to a specific deployed contract.
func NewProtocolFilterer(address common.Address, filterer bind.ContractFilterer) (*ProtocolFilterer, error) {
	contract, err := bindProtocol(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ProtocolFilterer{contract: contract}, nil
}

// bindProtocol binds a generic wrapper to an already deployed contract.
func bindProtocol(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ProtocolABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Protocol *ProtocolRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Protocol.Contract.ProtocolCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Protocol *ProtocolRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Protocol.Contract.ProtocolTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Protocol *ProtocolRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Protocol.Contract.ProtocolTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Protocol *ProtocolCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Protocol.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Protocol *ProtocolTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Protocol.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Protocol *ProtocolTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Protocol.Contract.contract.Transact(opts, method, params...)
}

// AddressesByUsername is a free data retrieval call binding the contract method 0xa58e325c.
//
// Solidity: function addressesByUsername( bytes32) constant returns(address)
func (_Protocol *ProtocolCaller) AddressesByUsername(opts *bind.CallOpts, arg0 [32]byte) (common.Address, error) {
	var (
		ret0 = new(common.Address)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "addressesByUsername", arg0)
	return *ret0, err
}

// AddressesByUsername is a free data retrieval call binding the contract method 0xa58e325c.
//
// Solidity: function addressesByUsername( bytes32) constant returns(address)
func (_Protocol *ProtocolSession) AddressesByUsername(arg0 [32]byte) (common.Address, error) {
	return _Protocol.Contract.AddressesByUsername(&_Protocol.CallOpts, arg0)
}

// AddressesByUsername is a free data retrieval call binding the contract method 0xa58e325c.
//
// Solidity: function addressesByUsername( bytes32) constant returns(address)
func (_Protocol *ProtocolCallerSession) AddressesByUsername(arg0 [32]byte) (common.Address, error) {
	return _Protocol.Contract.AddressesByUsername(&_Protocol.CallOpts, arg0)
}

// GetRefs is a free data retrieval call binding the contract method 0xb84da299.
//
// Solidity: function getRefs(repoID string, page uint256) constant returns(bytes)
func (_Protocol *ProtocolCaller) GetRefs(opts *bind.CallOpts, repoID string, page *big.Int) ([]byte, error) {
	var (
		ret0 = new([]byte)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "getRefs", repoID, page)
	return *ret0, err
}

// GetRefs is a free data retrieval call binding the contract method 0xb84da299.
//
// Solidity: function getRefs(repoID string, page uint256) constant returns(bytes)
func (_Protocol *ProtocolSession) GetRefs(repoID string, page *big.Int) ([]byte, error) {
	return _Protocol.Contract.GetRefs(&_Protocol.CallOpts, repoID, page)
}

// GetRefs is a free data retrieval call binding the contract method 0xb84da299.
//
// Solidity: function getRefs(repoID string, page uint256) constant returns(bytes)
func (_Protocol *ProtocolCallerSession) GetRefs(repoID string, page *big.Int) ([]byte, error) {
	return _Protocol.Contract.GetRefs(&_Protocol.CallOpts, repoID, page)
}

// NumRefs is a free data retrieval call binding the contract method 0xf2ebfa10.
//
// Solidity: function numRefs(repoID string) constant returns(uint256)
func (_Protocol *ProtocolCaller) NumRefs(opts *bind.CallOpts, repoID string) (*big.Int, error) {
	var (
		ret0 = new(*big.Int)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "numRefs", repoID)
	return *ret0, err
}

// NumRefs is a free data retrieval call binding the contract method 0xf2ebfa10.
//
// Solidity: function numRefs(repoID string) constant returns(uint256)
func (_Protocol *ProtocolSession) NumRefs(repoID string) (*big.Int, error) {
	return _Protocol.Contract.NumRefs(&_Protocol.CallOpts, repoID)
}

// NumRefs is a free data retrieval call binding the contract method 0xf2ebfa10.
//
// Solidity: function numRefs(repoID string) constant returns(uint256)
func (_Protocol *ProtocolCallerSession) NumRefs(repoID string) (*big.Int, error) {
	return _Protocol.Contract.NumRefs(&_Protocol.CallOpts, repoID)
}

// UserHasPullAccess is a free data retrieval call binding the contract method 0xede07dfe.
//
// Solidity: function userHasPullAccess(username string, repoID string) constant returns(bool)
func (_Protocol *ProtocolCaller) UserHasPullAccess(opts *bind.CallOpts, username string, repoID string) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "userHasPullAccess", username, repoID)
	return *ret0, err
}

// UserHasPullAccess is a free data retrieval call binding the contract method 0xede07dfe.
//
// Solidity: function userHasPullAccess(username string, repoID string) constant returns(bool)
func (_Protocol *ProtocolSession) UserHasPullAccess(username string, repoID string) (bool, error) {
	return _Protocol.Contract.UserHasPullAccess(&_Protocol.CallOpts, username, repoID)
}

// UserHasPullAccess is a free data retrieval call binding the contract method 0xede07dfe.
//
// Solidity: function userHasPullAccess(username string, repoID string) constant returns(bool)
func (_Protocol *ProtocolCallerSession) UserHasPullAccess(username string, repoID string) (bool, error) {
	return _Protocol.Contract.UserHasPullAccess(&_Protocol.CallOpts, username, repoID)
}

// UserHasPushAccess is a free data retrieval call binding the contract method 0x7bbaf159.
//
// Solidity: function userHasPushAccess(username string, repoID string) constant returns(bool)
func (_Protocol *ProtocolCaller) UserHasPushAccess(opts *bind.CallOpts, username string, repoID string) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "userHasPushAccess", username, repoID)
	return *ret0, err
}

// UserHasPushAccess is a free data retrieval call binding the contract method 0x7bbaf159.
//
// Solidity: function userHasPushAccess(username string, repoID string) constant returns(bool)
func (_Protocol *ProtocolSession) UserHasPushAccess(username string, repoID string) (bool, error) {
	return _Protocol.Contract.UserHasPushAccess(&_Protocol.CallOpts, username, repoID)
}

// UserHasPushAccess is a free data retrieval call binding the contract method 0x7bbaf159.
//
// Solidity: function userHasPushAccess(username string, repoID string) constant returns(bool)
func (_Protocol *ProtocolCallerSession) UserHasPushAccess(username string, repoID string) (bool, error) {
	return _Protocol.Contract.UserHasPushAccess(&_Protocol.CallOpts, username, repoID)
}

// UsernamesByAddress is a free data retrieval call binding the contract method 0x42dfb6da.
//
// Solidity: function usernamesByAddress( address) constant returns(string)
func (_Protocol *ProtocolCaller) UsernamesByAddress(opts *bind.CallOpts, arg0 common.Address) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "usernamesByAddress", arg0)
	return *ret0, err
}

// UsernamesByAddress is a free data retrieval call binding the contract method 0x42dfb6da.
//
// Solidity: function usernamesByAddress( address) constant returns(string)
func (_Protocol *ProtocolSession) UsernamesByAddress(arg0 common.Address) (string, error) {
	return _Protocol.Contract.UsernamesByAddress(&_Protocol.CallOpts, arg0)
}

// UsernamesByAddress is a free data retrieval call binding the contract method 0x42dfb6da.
//
// Solidity: function usernamesByAddress( address) constant returns(string)
func (_Protocol *ProtocolCallerSession) UsernamesByAddress(arg0 common.Address) (string, error) {
	return _Protocol.Contract.UsernamesByAddress(&_Protocol.CallOpts, arg0)
}

// CreateRepository is a paid mutator transaction binding the contract method 0xee94cf6d.
//
// Solidity: function createRepository(repoID string) returns()
func (_Protocol *ProtocolTransactor) CreateRepository(opts *bind.TransactOpts, repoID string) (*types.Transaction, error) {
	return _Protocol.contract.Transact(opts, "createRepository", repoID)
}

// CreateRepository is a paid mutator transaction binding the contract method 0xee94cf6d.
//
// Solidity: function createRepository(repoID string) returns()
func (_Protocol *ProtocolSession) CreateRepository(repoID string) (*types.Transaction, error) {
	return _Protocol.Contract.CreateRepository(&_Protocol.TransactOpts, repoID)
}

// CreateRepository is a paid mutator transaction binding the contract method 0xee94cf6d.
//
// Solidity: function createRepository(repoID string) returns()
func (_Protocol *ProtocolTransactorSession) CreateRepository(repoID string) (*types.Transaction, error) {
	return _Protocol.Contract.CreateRepository(&_Protocol.TransactOpts, repoID)
}

// SetUsername is a paid mutator transaction binding the contract method 0xed59313a.
//
// Solidity: function setUsername(username string) returns()
func (_Protocol *ProtocolTransactor) SetUsername(opts *bind.TransactOpts, username string) (*types.Transaction, error) {
	return _Protocol.contract.Transact(opts, "setUsername", username)
}

// SetUsername is a paid mutator transaction binding the contract method 0xed59313a.
//
// Solidity: function setUsername(username string) returns()
func (_Protocol *ProtocolSession) SetUsername(username string) (*types.Transaction, error) {
	return _Protocol.Contract.SetUsername(&_Protocol.TransactOpts, username)
}

// SetUsername is a paid mutator transaction binding the contract method 0xed59313a.
//
// Solidity: function setUsername(username string) returns()
func (_Protocol *ProtocolTransactorSession) SetUsername(username string) (*types.Transaction, error) {
	return _Protocol.Contract.SetUsername(&_Protocol.TransactOpts, username)
}

// UpdateRef is a paid mutator transaction binding the contract method 0x605a5dd8.
//
// Solidity: function updateRef(repoID string, refName string, commitHash string) returns()
func (_Protocol *ProtocolTransactor) UpdateRef(opts *bind.TransactOpts, repoID string, refName string, commitHash string) (*types.Transaction, error) {
	return _Protocol.contract.Transact(opts, "updateRef", repoID, refName, commitHash)
}

// UpdateRef is a paid mutator transaction binding the contract method 0x605a5dd8.
//
// Solidity: function updateRef(repoID string, refName string, commitHash string) returns()
func (_Protocol *ProtocolSession) UpdateRef(repoID string, refName string, commitHash string) (*types.Transaction, error) {
	return _Protocol.Contract.UpdateRef(&_Protocol.TransactOpts, repoID, refName, commitHash)
}

// UpdateRef is a paid mutator transaction binding the contract method 0x605a5dd8.
//
// Solidity: function updateRef(repoID string, refName string, commitHash string) returns()
func (_Protocol *ProtocolTransactorSession) UpdateRef(repoID string, refName string, commitHash string) (*types.Transaction, error) {
	return _Protocol.Contract.UpdateRef(&_Protocol.TransactOpts, repoID, refName, commitHash)
}

// ProtocolLogSetUsernameIterator is returned from FilterLogSetUsername and is used to iterate over the raw logs and unpacked data for LogSetUsername events raised by the Protocol contract.
type ProtocolLogSetUsernameIterator struct {
	Event *ProtocolLogSetUsername // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ProtocolLogSetUsernameIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ProtocolLogSetUsername)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ProtocolLogSetUsername)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ProtocolLogSetUsernameIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ProtocolLogSetUsernameIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ProtocolLogSetUsername represents a LogSetUsername event raised by the Protocol contract.
type ProtocolLogSetUsername struct {
	Addr     common.Address
	Username string
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterLogSetUsername is a free log retrieval operation binding the contract event 0xaffa6dd92f7ba89dd7b4fdd8809b8e8d38b6431d8f41674fae86cfa06fc66d99.
//
// Solidity: e LogSetUsername(addr address, username string)
func (_Protocol *ProtocolFilterer) FilterLogSetUsername(opts *bind.FilterOpts) (*ProtocolLogSetUsernameIterator, error) {

	logs, sub, err := _Protocol.contract.FilterLogs(opts, "LogSetUsername")
	if err != nil {
		return nil, err
	}
	return &ProtocolLogSetUsernameIterator{contract: _Protocol.contract, event: "LogSetUsername", logs: logs, sub: sub}, nil
}

// WatchLogSetUsername is a free log subscription operation binding the contract event 0xaffa6dd92f7ba89dd7b4fdd8809b8e8d38b6431d8f41674fae86cfa06fc66d99.
//
// Solidity: e LogSetUsername(addr address, username string)
func (_Protocol *ProtocolFilterer) WatchLogSetUsername(opts *bind.WatchOpts, sink chan<- *ProtocolLogSetUsername) (event.Subscription, error) {

	logs, sub, err := _Protocol.contract.WatchLogs(opts, "LogSetUsername")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ProtocolLogSetUsername)
				if err := _Protocol.contract.UnpackLog(event, "LogSetUsername", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}
