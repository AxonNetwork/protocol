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
const ProtocolABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"usernamesByAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"refName\",\"type\":\"string\"},{\"name\":\"commitHash\",\"type\":\"string\"}],\"name\":\"updateRef\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPushAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"addressesByUsername\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"page\",\"type\":\"uint256\"}],\"name\":\"getRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"}],\"name\":\"setUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPullAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"createRepository\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"numRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"username\",\"type\":\"string\"}],\"name\":\"LogSetUsername\",\"type\":\"event\"}]"

// ProtocolBin is the compiled bytecode used for deploying new contracts.
const ProtocolBin = `0x608060405234801561001057600080fd5b50611425806100206000396000f3006080604052600436106100985763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166342dfb6da811461009d578063605a5dd8146101405780637bbaf15914610217578063a58e325c146102c2578063b84da29914610303578063ed59313a1461035e578063ede07dfe146103b7578063ee94cf6d1461044e578063f2ebfa10146104a7575b600080fd5b3480156100a957600080fd5b506100cb73ffffffffffffffffffffffffffffffffffffffff60043516610512565b6040805160208082528351818301528351919283929083019185019080838360005b838110156101055781810151838201526020016100ed565b50505050905090810190601f1680156101325780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561014c57600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261021594369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506105ac9650505050505050565b005b34801561022357600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102ae94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375094975061070a9650505050505050565b604080519115158252519081900360200190f35b3480156102ce57600080fd5b506102da6004356108b9565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b34801561030f57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526100cb94369492936024939284019190819084018382808284375094975050933594506108e19350505050565b34801561036a57600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610215943694929360249392840191908190840183828082843750949750610d6a9650505050505050565b3480156103c357600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102ae94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a999881019791965091820194509250829150840183828082843750949750610ed99650505050505050565b34801561045a57600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610215943694929360249392840191908190840183828082843750949750610f419650505050505050565b3480156104b357600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526105009436949293602493928401919081908401838280828437509497506110a59650505050505050565b60408051918252519081900360200190f35b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152929091908301828280156105a45780601f10610579576101008083540402835291602001916105a4565b820191906000526020600020905b81548152906001019060200180831161058757829003601f168201915b505050505081565b336000908152602081815260408083208054825160026001831615610100026000190190921691909104601f81018590048502820185019093528281528493849361064e93918301828280156106435780601f1061061857610100808354040283529160200191610643565b820191906000526020600020905b81548152906001019060200180831161062657829003601f168201915b50505050508761070a565b151561065957600080fd5b610662866110cb565b6000818152600260205260409020909350915061067e856110cb565b6000818152600180850160205260409091205491925060026101009183161591909102600019019091160415156106e057600282018054600181018083556000928352602092839020885191936106dd93919091019190890190611347565b50505b60008181526001830160209081526040909120855161070192870190611347565b50505050505050565b6000806000836040516020018082805190602001908083835b602083106107425780518252601f199092019160209182019101610723565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106107a55780518252601f199092019160209182019101610786565b51815160209384036101000a60001901801990921691161790526040519190930181900381208a519097508a955090830193508392850191508083835b602083106108015780518252601f1990920191602091820191016107e2565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106108645780518252601f199092019160209182019101610845565b51815160209384036101000a600019018019909216911617905260408051929094018290039091206000888152600283528481208282526003019092529290205460ff1696509093505050505b505092915050565b60016020526000908152604090205473ffffffffffffffffffffffffffffffffffffffff1681565b606060008060608060008060006060806108fa8c6110cb565b60008181526002602052604090208054919a50985060ff16151561091d57600080fd5b60408051600a808252610160820190925290816020015b606081526020019060019003908161093457505060408051600a8082526101608201909252919850602082015b60608152602001906001900390816109615790505095508a600a029450600093505b60028801548486011015610bc357610a3f886002018686018154811015156109a757fe5b600091825260209182902001805460408051601f6002600019610100600187161502019094169390930492830185900485028101850190915281815292830182828015610a355780601f10610a0a57610100808354040283529160200191610a35565b820191906000526020600020905b815481529060010190602001808311610a1857829003601f168201915b50505050506110cb565b925087600201858501815481101515610a5457fe5b600091825260209182902001805460408051601f6002600019610100600187161502019094169390930492830185900485028101850190915281815292830182828015610ae25780601f10610ab757610100808354040283529160200191610ae2565b820191906000526020600020905b815481529060010190602001808311610ac557829003601f168201915b50505050508785815181101515610af557fe5b602090810290910181019190915260008481526001808b01835260409182902080548351601f6002610100958416159590950260001901909216939093049081018590048502830185019093528282529092909190830182828015610b9b5780601f10610b7057610100808354040283529160200191610b9b565b820191906000526020600020905b815481529060010190602001808311610b7e57829003601f168201915b50505050508685815181101515610bae57fe5b60209081029091010152600190930192610983565b60408051600a808252610160820190925290816020015b610be26113c5565b815260200190600190039081610bda579050509150600093505b600a841015610d18576040805160028082526060820190925290816020015b610c236113c5565b815260200190600190039081610c1b579050509050610c588785815181101515610c4957fe5b90602001906020020151611195565b816000815181101515610c6757fe5b602090810290910101528551610c8390879086908110610c4957fe5b816001815181101515610c9257fe5b90602001906020020181905250610cf5610cf082610ce46040805190810160405280600181526020017f3a00000000000000000000000000000000000000000000000000000000000000815250611195565b9063ffffffff6111bb16565b611195565b8285815181101515610d0357fe5b60209081029091010152600190930192610bfc565b610d5a82610ce46040805190810160405280600181526020017f2f00000000000000000000000000000000000000000000000000000000000000815250611195565b9c9b505050505050505050505050565b6000808251111515610d7b57600080fd5b336000908152602081905260409020546002600019610100600184161502019091160415610da857600080fd5b610db1826110cb565b60008181526001602052604090205490915073ffffffffffffffffffffffffffffffffffffffff1615610de357600080fd5b336000908152602081815260409091208351610e0192850190611347565b506000818152600160209081526040808320805473ffffffffffffffffffffffffffffffffffffffff191633908117909155815181815280840183815287519382019390935286517faffa6dd92f7ba89dd7b4fdd8809b8e8d38b6431d8f41674fae86cfa06fc66d9995929488949293606085019291860191908190849084905b83811015610e9a578181015183820152602001610e82565b50505050905090810190601f168015610ec75780820380516001836020036101000a031916815260200191505b50935050505060405180910390a15050565b6000806000610ee7846110cb565b60008181526002602052604090206005015490925060ff161515610f0e57600192506108b1565b610f17856110cb565b6000928352600260209081526040808520928552600690920190529091205460ff16949350505050565b606060008060008451111515610f5657600080fd5b336000908152602081815260409182902080548351601f600260001961010060018616150201909316929092049182018490048402810184019094528084529091830182828015610fe85780601f10610fbd57610100808354040283529160200191610fe8565b820191906000526020600020905b815481529060010190602001808311610fcb57829003601f168201915b5050505050925060008351111515610fff57600080fd5b611008846110cb565b9150611013836110cb565b60008381526002602052604090205490915060ff161561103257600080fd5b60008281526002602090815260408220805460ff1916600190811782556004909101805491820180825590845292829020865161107793919092019190870190611347565b50506000918252600260209081526040808420928452600390920190529020805460ff191660011790555050565b6000806110b1836110cb565b600090815260026020819052604090912001549392505050565b6000816040516020018082805190602001908083835b602083106111005780518252601f1990920191602091820191016110e1565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106111635780518252601f199092019160209182019101611144565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b61119d6113c5565b50604080518082019091528151815260209182019181019190915290565b6060600080606060008551600014156111e45760408051602081019091526000815294506112f9565b60018651038760000151029350600092505b855183101561122857858381518110151561120d57fe5b602090810290910101515193909301926001909201916111f6565b836040519080825280601f01601f191660200182016040528015611256578160200160208202803883390190505b5060009350915050602081015b85518310156112f5576112aa81878581518110151561127e57fe5b9060200190602002015160200151888681518110151561129a57fe5b6020908102909101015151611303565b85838151811015156112b857fe5b60209081029091010151518651910190600019018310156112ea576112e68188602001518960000151611303565b8651015b600190920191611263565b8194505b5050505092915050565b60005b60208210611328578251845260209384019390920191601f1990910190611306565b50905182516020929092036101000a6000190180199091169116179052565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061138857805160ff19168380011785556113b5565b828001600101855582156113b5579182015b828111156113b557825182559160200191906001019061139a565b506113c19291506113dc565b5090565b604080518082019091526000808252602082015290565b6113f691905b808211156113c157600081556001016113e2565b905600a165627a7a7230582011ea76f8c609028b608d81cd9e5d7ea28e6a9c15bb27762bc4401f2a7e5bade70029`

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
// Solidity: function getRefs(repoID string, page uint256) constant returns(string)
func (_Protocol *ProtocolCaller) GetRefs(opts *bind.CallOpts, repoID string, page *big.Int) (string, error) {
	var (
		ret0 = new(string)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "getRefs", repoID, page)
	return *ret0, err
}

// GetRefs is a free data retrieval call binding the contract method 0xb84da299.
//
// Solidity: function getRefs(repoID string, page uint256) constant returns(string)
func (_Protocol *ProtocolSession) GetRefs(repoID string, page *big.Int) (string, error) {
	return _Protocol.Contract.GetRefs(&_Protocol.CallOpts, repoID, page)
}

// GetRefs is a free data retrieval call binding the contract method 0xb84da299.
//
// Solidity: function getRefs(repoID string, page uint256) constant returns(string)
func (_Protocol *ProtocolCallerSession) GetRefs(repoID string, page *big.Int) (string, error) {
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

// StringsABI is the input ABI used to generate the binding from.
const StringsABI = "[]"

// StringsBin is the compiled bytecode used for deploying new contracts.
const StringsBin = `0x604c602c600b82828239805160001a60731460008114601c57601e565bfe5b5030600052607381538281f30073000000000000000000000000000000000000000030146080604052600080fd00a165627a7a72305820b9f9acbf069371e129c33b464ac525ff0bd8909aa15271b7aa68d3d20d831b370029`

// DeployStrings deploys a new Ethereum contract, binding an instance of Strings to it.
func DeployStrings(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Strings, error) {
	parsed, err := abi.JSON(strings.NewReader(StringsABI))
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(StringsBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Strings{StringsCaller: StringsCaller{contract: contract}, StringsTransactor: StringsTransactor{contract: contract}, StringsFilterer: StringsFilterer{contract: contract}}, nil
}

// Strings is an auto generated Go binding around an Ethereum contract.
type Strings struct {
	StringsCaller     // Read-only binding to the contract
	StringsTransactor // Write-only binding to the contract
	StringsFilterer   // Log filterer for contract events
}

// StringsCaller is an auto generated read-only Go binding around an Ethereum contract.
type StringsCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StringsTransactor is an auto generated write-only Go binding around an Ethereum contract.
type StringsTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StringsFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type StringsFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// StringsSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type StringsSession struct {
	Contract     *Strings          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// StringsCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type StringsCallerSession struct {
	Contract *StringsCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// StringsTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type StringsTransactorSession struct {
	Contract     *StringsTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// StringsRaw is an auto generated low-level Go binding around an Ethereum contract.
type StringsRaw struct {
	Contract *Strings // Generic contract binding to access the raw methods on
}

// StringsCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type StringsCallerRaw struct {
	Contract *StringsCaller // Generic read-only contract binding to access the raw methods on
}

// StringsTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type StringsTransactorRaw struct {
	Contract *StringsTransactor // Generic write-only contract binding to access the raw methods on
}

// NewStrings creates a new instance of Strings, bound to a specific deployed contract.
func NewStrings(address common.Address, backend bind.ContractBackend) (*Strings, error) {
	contract, err := bindStrings(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Strings{StringsCaller: StringsCaller{contract: contract}, StringsTransactor: StringsTransactor{contract: contract}, StringsFilterer: StringsFilterer{contract: contract}}, nil
}

// NewStringsCaller creates a new read-only instance of Strings, bound to a specific deployed contract.
func NewStringsCaller(address common.Address, caller bind.ContractCaller) (*StringsCaller, error) {
	contract, err := bindStrings(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &StringsCaller{contract: contract}, nil
}

// NewStringsTransactor creates a new write-only instance of Strings, bound to a specific deployed contract.
func NewStringsTransactor(address common.Address, transactor bind.ContractTransactor) (*StringsTransactor, error) {
	contract, err := bindStrings(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &StringsTransactor{contract: contract}, nil
}

// NewStringsFilterer creates a new log filterer instance of Strings, bound to a specific deployed contract.
func NewStringsFilterer(address common.Address, filterer bind.ContractFilterer) (*StringsFilterer, error) {
	contract, err := bindStrings(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &StringsFilterer{contract: contract}, nil
}

// bindStrings binds a generic wrapper to an already deployed contract.
func bindStrings(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(StringsABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Strings *StringsRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Strings.Contract.StringsCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Strings *StringsRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Strings.Contract.StringsTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Strings *StringsRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Strings.Contract.StringsTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Strings *StringsCallerRaw) Call(opts *bind.CallOpts, result interface{}, method string, params ...interface{}) error {
	return _Strings.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Strings *StringsTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Strings.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Strings *StringsTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Strings.Contract.contract.Transact(opts, method, params...)
}
