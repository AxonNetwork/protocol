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
const ProtocolABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"usernamesByAddress\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"user\",\"type\":\"address\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"addressHasPullAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"refName\",\"type\":\"string\"},{\"name\":\"commitHash\",\"type\":\"string\"}],\"name\":\"updateRef\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPushAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"bytes32\"}],\"name\":\"addressesByUsername\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"repositoryExists\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"},{\"name\":\"page\",\"type\":\"uint256\"}],\"name\":\"getRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"}],\"name\":\"setUsername\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"username\",\"type\":\"string\"},{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"userHasPullAccess\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"createRepository\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"repoID\",\"type\":\"string\"}],\"name\":\"numRefs\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"addr\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"username\",\"type\":\"string\"}],\"name\":\"LogSetUsername\",\"type\":\"event\"}]"

// ProtocolBin is the compiled bytecode used for deploying new contracts.
const ProtocolBin = `0x608060405234801561001057600080fd5b5061143d806100206000396000f3006080604052600436106100ae5763ffffffff7c010000000000000000000000000000000000000000000000000000000060003504166342dfb6da81146100b35780634d6e93cb14610156578063605a5dd8146101de5780637bbaf159146102b5578063a58e325c1461034c578063ae268e301461038d578063b84da299146103e6578063ed59313a14610441578063ede07dfe1461049a578063ee94cf6d14610531578063f2ebfa101461058a575b600080fd5b3480156100bf57600080fd5b506100e173ffffffffffffffffffffffffffffffffffffffff600435166105f5565b6040805160208082528351818301528351919283929083019185019080838360005b8381101561011b578181015183820152602001610103565b50505050905090810190601f1680156101485780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561016257600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101ca95833573ffffffffffffffffffffffffffffffffffffffff1695369560449491939091019190819084018382808284375094975061068f9650505050505050565b604080519115158252519081900360200190f35b3480156101ea57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102b394369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506107589650505050505050565b005b3480156102c157600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101ca94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a9998810197919650918201945092508291508401838280828437509497506108b69650505050505050565b34801561035857600080fd5b50610364600435610a5e565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b34801561039957600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101ca943694929360249392840191908190840183828082843750949750610a869650505050505050565b3480156103f257600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526100e19436949293602493928401919081908401838280828437509497505093359450610abb9350505050565b34801561044d57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102b3943694929360249392840191908190840183828082843750949750610e619650505050505050565b3480156104a657600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101ca94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a999881019791965091820194509250829150840183828082843750949750610fd09650505050505050565b34801561053d57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526102b39436949293602493928401919081908401838280828437509497506110389650505050505050565b34801561059657600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526105e394369492936024939284019190819084018382808284375094975061119c9650505050505050565b60408051918252519081900360200190f35b600060208181529181526040908190208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152929091908301828280156106875780601f1061065c57610100808354040283529160200191610687565b820191906000526020600020905b81548152906001019060200180831161066a57829003601f168201915b505050505081565b73ffffffffffffffffffffffffffffffffffffffff82166000908152602081815260408083208054825160026001831615610100026000190190921691909104601f81018590048502820185019093528281529092849261074a929185919083018282801561073f5780601f106107145761010080835404028352916020019161073f565b820191906000526020600020905b81548152906001019060200180831161072257829003601f168201915b505050505085610fd0565b90508092505b505092915050565b336000908152602081815260408083208054825160026001831615610100026000190190921691909104601f8101859004850282018501909352828152849384936107fa93918301828280156107ef5780601f106107c4576101008083540402835291602001916107ef565b820191906000526020600020905b8154815290600101906020018083116107d257829003601f168201915b5050505050876108b6565b151561080557600080fd5b61080e866111c2565b6000818152600260205260409020909350915061082a856111c2565b60008181526001808501602052604090912054919250600261010091831615919091026000190190911604151561088c576002820180546001810180835560009283526020928390208851919361088993919091019190890190611376565b50505b6000818152600183016020908152604090912085516108ad92870190611376565b50505050505050565b6000806000836040516020018082805190602001908083835b602083106108ee5780518252601f1990920191602091820191016108cf565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b602083106109515780518252601f199092019160209182019101610932565b51815160209384036101000a60001901801990921691161790526040519190930181900381208a519097508a955090830193508392850191508083835b602083106109ad5780518252601f19909201916020918201910161098e565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b60208310610a105780518252601f1990920191602091820191016109f1565b51815160209384036101000a600019018019909216911617905260408051929094018290039091206000978852600282528388209088526003019052509093205460ff169695505050505050565b60016020526000908152604090205473ffffffffffffffffffffffffffffffffffffffff1681565b60008060008351111515610a9957600080fd5b610aa2836111c2565b60009081526002602052604090205460ff169392505050565b606060008060008060008060008060606000610ad68d6111c2565b60008181526002602052604090208054919b50995060ff161515610af957600080fd5b600094508b600a029350600092505b60028901548385011015610c4857610bc489600201858501815481101515610b2c57fe5b600091825260209182902001805460408051601f6002600019610100600187161502019094169390930492830185900485028101850190915281815292830182828015610bba5780601f10610b8f57610100808354040283529160200191610bba565b820191906000526020600020905b815481529060010190602001808311610b9d57829003601f168201915b50505050506111c2565b955088600201848401815481101515610bd957fe5b90600052602060002001975088600101600087600019166000191681526020019081526020016000209650868054600181600116156101000203166002900490508880546001816001161561010002031660029004905060200160200101850194508280600101935050610b08565b846040519080825280601f01601f191660200182016040528015610c76578160200160208202803883390190505b50915060009050600092505b60028901548385011015610e5157610ca689600201858501815481101515610b2c57fe5b955088600201848401815481101515610cbb57fe5b600091825260208083208984526001808e019092526040909320929091018054909a50919850610cff9160026101009282161592909202600019011604838361128c565b875460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292820192610d999290918b91830182828015610d8d5780601f10610d6257610100808354040283529160200191610d8d565b820191906000526020600020905b815481529060010190602001808311610d7057829003601f168201915b505050505083836112fb565b875487546002600019610100600180861615820283019095168390049590950194610dce94841615020190911604838361128c565b865460408051602060026001851615610100026000190190941693909304601f810184900484028201840190925281815292820192610e319290918a91830182828015610d8d5780601f10610d6257610100808354040283529160200191610d8d565b865460019384019360029082161561010002600019019091160401610c82565b509b9a5050505050505050505050565b6000808251111515610e7257600080fd5b336000908152602081905260409020546002600019610100600184161502019091160415610e9f57600080fd5b610ea8826111c2565b60008181526001602052604090205490915073ffffffffffffffffffffffffffffffffffffffff1615610eda57600080fd5b336000908152602081815260409091208351610ef892850190611376565b506000818152600160209081526040808320805473ffffffffffffffffffffffffffffffffffffffff191633908117909155815181815280840183815287519382019390935286517faffa6dd92f7ba89dd7b4fdd8809b8e8d38b6431d8f41674fae86cfa06fc66d9995929488949293606085019291860191908190849084905b83811015610f91578181015183820152602001610f79565b50505050905090810190601f168015610fbe5780820380516001836020036101000a031916815260200191505b50935050505060405180910390a15050565b6000806000610fde846111c2565b60008181526002602052604090206005015490925060ff1615156110055760019250610750565b61100e856111c2565b6000928352600260209081526040808520928552600690920190529091205460ff16949350505050565b60606000806000845111151561104d57600080fd5b336000908152602081815260409182902080548351601f6002600019610100600186161502019093169290920491820184900484028101840190945280845290918301828280156110df5780601f106110b4576101008083540402835291602001916110df565b820191906000526020600020905b8154815290600101906020018083116110c257829003601f168201915b50505050509250600083511115156110f657600080fd5b6110ff846111c2565b915061110a836111c2565b60008381526002602052604090205490915060ff161561112957600080fd5b60008281526002602090815260408220805460ff1916600190811782556004909101805491820180825590845292829020865161116e93919092019190870190611376565b50506000918252600260209081526040808420928452600390920190529020805460ff191660011790555050565b6000806111a8836111c2565b600090815260026020819052604090912001549392505050565b6000816040516020018082805190602001908083835b602083106111f75780518252601f1990920191602091820191016111d8565b6001836020036101000a0380198251168184511680821785525050505050509050019150506040516020818303038152906040526040518082805190602001908083835b6020831061125a5780518252601f19909201916020918201910161123b565b5181516020939093036101000a6000190180199091169216919091179052604051920182900390912095945050505050565b8260005b60208110156112f4578181602081106112a557fe5b1a60f860020a02848483018151811015156112bc57fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a905350600101611290565b5050505050565b60005b835181101561137057838181518110151561131557fe5b90602001015160f860020a900460f860020a028383830181518110151561133857fe5b9060200101907effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff1916908160001a9053506001016112fe565b50505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106113b757805160ff19168380011785556113e4565b828001600101855582156113e4579182015b828111156113e45782518255916020019190600101906113c9565b506113f09291506113f4565b5090565b61140e91905b808211156113f057600081556001016113fa565b905600a165627a7a723058203f60143c5d47990dbc68bc5f80a781d05dc74f80304e1a6d1daf01019c2a9e640029`

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

// AddressHasPullAccess is a free data retrieval call binding the contract method 0x4d6e93cb.
//
// Solidity: function addressHasPullAccess(user address, repoID string) constant returns(bool)
func (_Protocol *ProtocolCaller) AddressHasPullAccess(opts *bind.CallOpts, user common.Address, repoID string) (bool, error) {
	var (
		ret0 = new(bool)
	)
	out := ret0
	err := _Protocol.contract.Call(opts, out, "addressHasPullAccess", user, repoID)
	return *ret0, err
}

// AddressHasPullAccess is a free data retrieval call binding the contract method 0x4d6e93cb.
//
// Solidity: function addressHasPullAccess(user address, repoID string) constant returns(bool)
func (_Protocol *ProtocolSession) AddressHasPullAccess(user common.Address, repoID string) (bool, error) {
	return _Protocol.Contract.AddressHasPullAccess(&_Protocol.CallOpts, user, repoID)
}

// AddressHasPullAccess is a free data retrieval call binding the contract method 0x4d6e93cb.
//
// Solidity: function addressHasPullAccess(user address, repoID string) constant returns(bool)
func (_Protocol *ProtocolCallerSession) AddressHasPullAccess(user common.Address, repoID string) (bool, error) {
	return _Protocol.Contract.AddressHasPullAccess(&_Protocol.CallOpts, user, repoID)
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
