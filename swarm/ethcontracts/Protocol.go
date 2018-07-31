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
const ProtocolABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"usernamesByAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"refName\",\"type\":\"string\"},{\"name\":\"commitHash\",\"type\":\"string\"}],\"name\":\"updateRef\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPushAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"addressesByUsername\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"repositoryExists\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"page\",\"type\":\"uint256\"}],\"name\":\"getRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"}],\"name\":\"setUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPullAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"createRepository\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"numRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"username\",\"type\":\"string\"}],\"name\":\"LogSetUsername\",\"type\":\"event\"}]"

// ProtocolBin is the compiled bytecode used for deploying new contracts.
const ProtocolBin = `0x608060405234801561001057600080fd5b506112fc806100206000396000f3006080604052600436106100a35763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166342dfb6da81146100a8578063605a5dd81461014b5780637bbaf15914610222578063a58e325c146102cd578063ae268e301461030e578063b84da29914610367578063ed59313a146103c2578063ede07dfe1461041b578063ee94cf6d146104b2578063f2ebfa101461050b575b600080fd5b3480156100b457600080fd5b506100d673ffffffffffffffffffffffffffffffffffffffff60043516610576565b6040805160208082528351818301528351919283929083019185019080838360005b838110156101105781810151838201526020016100f8565b50505050905090810190601f16801561013d5780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561015757600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261022094369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506106109650505050505050565b005b34801561022e57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102b994369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375094975061076e9650505050505050565b604080519115158252519081900360200190f35b3480156102d957600080fd5b506102e560043561091d565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b34801561031a57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102b99436949293602493928401919081908401838280828437509497506109459650505050505050565b34801561037357600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526100d6943694929360249392840191908190840183828082843750949750509335945061097a9350505050565b3480156103ce57600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610220943694929360249392840191908190840183828082843750949750610d209650505050505050565b34801561042757600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102b994369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a999881019791965091820194509250829150840183828082843750949750610e8f9650505050505050565b3480156104be57600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610220943694929360249392840191908190840183828082843750949750610ef79650505050505050565b34801561051757600080fd5b506040805160206004803580820135601f810184900484028501840190955284845261056494369492936024939284019190819084018382808284375094975061105b9650505050505050565b60408051918252519081900360200190f35b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152929091908301828280156106085780601f106105dd57610100808354040283529160200191610608565b820191906000526020600020905b8154815290600101906020018083116105eb57829003601f168201915b505050505081565b336000908152602081815260408083208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152849384936106b293918301828280156106a75780601f1061067c576101008083540402835291602001916106a7565b820191906000526020600020905b81548152906001019060200180831161068a57829003601f168201915b50505050508761076e565b15156106bd57600080fd5b6106c686611081565b600081815260026020526040902090935091506106e285611081565b600081815260018085016020526040909120549192506002610100918316159190910260001901909116041515610744576002820180546001810180835560009283526020928390208851919361074193919091019190890190611235565b50505b60008181526001830160209081526040909120855161076592870190611235565b50505050505050565b6000806000836040516020018082805190602001908083835b602083106107a65780518252601f199092019160209182019101610787565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106108095780518252601f1990920191602091820191016107ea565b51815160209384036101000a60001901801990921691161790526040519190930181900381208a519097508a955090830193508392850191508083835b602083106108655780518252601f199092019160209182019101610846565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106108c85780518252601f1990920191602091820191016108a9565b51815160209384036101000a600019018019909216911617905260408051929094018290039091206000888152600283528481208282526003019092529290205460ff1696509093505050505b505092915050565b60016020526000908152604090205473ffffffffffffffffffffffffffffffffffffffff1681565b6000806000835111151561095857600080fd5b61096183611081565b60009081526002602052604090205460ff169392505050565b6060600080600080600080600080606060006109958d611081565b60008181526002602052604090208054919b50995060ff1615156109b857600080fd5b600094508b600a029350600092505b60028901548385011015610b0757610a83896002018585018154811015156109eb57fe5b600091825260209182902001805460408051601f6002600019610100600187161502019094169390930492830185900485028101850190915281815292830182828015610a795780601f10610a4e57610100808354040283529160200191610a79565b820191906000526020600020905b815481529060010190602001808311610a5c57829003601f168201915b5050505050611081565b955088600201848401815481101515610a9857fe5b906000526020600020019750886001016000876000191660001916815260200190815260200160002096508680546001816001161561010002031660029004905088805460018160011615610100020316600290049050602001602001018501945082806001019350506109c7565b846040519080825280601f01601f191660200182016040528015610b35578160200160208202803883390190505b50915060009050600092505b60028901548385011015610d1057610b65896002018585018154811015156109eb57fe5b955088600201848401815481101515610b7a57fe5b600091825260208083208984526001808e019092526040909320929091018054909a50919850610bbe9160026101009282161592909202600019011604838361114b565b875460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292820192610c589290918b91830182828015610c4c5780601f10610c2157610100808354040283529160200191610c4c565b820191906000526020600020905b815481529060010190602001808311610c2f57829003601f168201915b505050505083836111ba565b875487546002600019610100600180861615820283019095168390049590950194610c8d94841615020190911604838361114b565b865460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292820192610cf09290918a91830182828015610c4c5780601f10610c2157610100808354040283529160200191610c4c565b865460019384019360029082161561010002600019019091160401610b41565b509b9a5050505050505050505050565b6000808251111515610d3157600080fd5b336000908152602081905260409020546002600019610100600184161502019091160415610d5e57600080fd5b610d6782611081565b60008181526001602052604090205490915073ffffffffffffffffffffffffffffffffffffffff1615610d9957600080fd5b336000908152602081815260409091208351610db792850190611235565b506000818152600160209081526040808320805473ffffffffffffffffffffffffffffffffffffffff191633908117909155815181815280840183815287519382019390935286517faffa6dd92f7ba89dd7b4fdd8809b8e8d38b6431d8f41674fae86cfa06fc66d9995929488949293606085019291860191908190849084905b83811015610e50578181015183820152602001610e38565b50505050905090810190601f168015610e7d5780820380516001836020036101000a031916815260200191505b50935050505060405180910390a15050565b6000806000610e9d84611081565b60008181526002602052604090206005015490925060ff161515610ec45760019250610915565b610ecd85611081565b6000928352600260209081526040808520928552600690920190529091205460ff16949350505050565b606060008060008451111515610f0c57600080fd5b336000908152602081815260409182902080548351601f600260001961010060018616150201909316929092049182018490048402810184019094528084529091830182828015610f9e5780601f10610f7357610100808354040283529160200191610f9e565b820191906000526020600020905b815481529060010190602001808311610f8157829003601f168201915b5050505050925060008351111515610fb557600080fd5b610fbe84611081565b9150610fc983611081565b60008381526002602052604090205490915060ff1615610fe857600080fd5b60008281526002602090815260408220805460ff1916600190811782556004909101805491820180825590845292829020865161102d93919092019190870190611235565b50506000918252600260209081526040808420928452600390920190529020805460ff191660011790555050565b60008061106783611081565b600090815260026020819052604090912001549392505050565b6000816040516020018082805190602001908083835b602083106110b65780518252601f199092019160209182019101611097565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106111195780518252601f1990920191602091820191016110fa565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b8260005b60208110156111b35781816020811061116457fe5b1a60f860020a028484830181518110151561117b57fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a90535060010161114f565b5050505050565b60005b835181101561122f5783818151811015156111d457fe5b90602001015160f860020a900460f860020a02838383018151811015156111f757fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506001016111bd565b50505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f1061127657805160ff19168380011785556112a3565b828001600101855582156112a3579182015b828111156112a3578251825591602001919060010190611288565b506112af9291506112b3565b5090565b6112cd91905b808211156112af57600081556001016112b9565b905600a165627a7a72305820d30e6e3c82a2c401c3a423ab13b6de74c1b6143df219cf523c8d9e9b8f4c05030029`

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

// RepositoryExists is a free data retrieval call binding the contract method 0xae268e30.
//
// Solidity: function repositoryExists(repoID string) constant returns(bool)
func (_Protocol *ProtocolCaller) RepositoryExists(opts *bind.CallOpts, repoID string) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "repositoryExists", repoID)
	return *ret0, err
}

// RepositoryExists is a free data retrieval call binding the contract method 0xae268e30.
//
// Solidity: function repositoryExists(repoID string) constant returns(bool)
func (_Protocol *ProtocolSession) RepositoryExists(repoID string) (bool, error) {
	return _Protocol.Contract.RepositoryExists(&_Protocol.CallOpts, repoID)
}

// RepositoryExists is a free data retrieval call binding the contract method 0xae268e30.
//
// Solidity: function repositoryExists(repoID string) constant returns(bool)
func (_Protocol *ProtocolCallerSession) RepositoryExists(repoID string) (bool, error) {
	return _Protocol.Contract.RepositoryExists(&_Protocol.CallOpts, repoID)
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
