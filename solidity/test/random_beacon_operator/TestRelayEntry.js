const blsData = require("../helpers/data.js")
const initContracts = require('../helpers/initContracts')
const assert = require('chai').assert
const expectThrow = require('../helpers/expectThrow.js')
const expectThrowWithMessage = require('../helpers/expectThrowWithMessage.js')
const {createSnapshot, restoreSnapshot} = require("../helpers/snapshot.js")
const {contract, accounts, web3} = require("@openzeppelin/test-environment")

describe('KeepRandomBeaconOperator/RelayEntry', () => {
  let serviceContract, operatorContract;

  before(async () => {

    let contracts = await initContracts(
      contract.fromArtifact('KeepToken'),
      contract.fromArtifact('TokenStaking'),
      contract.fromArtifact('KeepRandomBeaconService'),
      contract.fromArtifact('KeepRandomBeaconServiceImplV1'),
      contract.fromArtifact('KeepRandomBeaconOperatorStub')
    );

    operatorContract = contracts.operatorContract;
    serviceContract = contracts.serviceContract;

    // Using stub method to add first group to help testing.
    await operatorContract.registerNewGroup(blsData.groupPubKey);
    operatorContract.setGroupSize(3);
    let group = await operatorContract.getGroupPublicKey(0);
    await operatorContract.setGroupMembers(group, [accounts[0], accounts[1], accounts[2]]);

    let entryFeeEstimate = await serviceContract.entryFeeEstimate(0);
    await serviceContract.methods['requestRelayEntry()']({value: entryFeeEstimate});
  });

  beforeEach(async () => {
    await createSnapshot()
  });

  afterEach(async () => {
    await restoreSnapshot()
  });

  it("should keep relay entry submission at reasonable price", async () => {
    let gasEstimate = await operatorContract.relayEntry.estimateGas(blsData.groupSignature);

    // Make sure no change will make the verification more expensive than it is
    // now or that even if it happens, it will be a conscious decision.
    assert.isBelow(gasEstimate, 378902, "Relay entry submission is too expensive")
  });

  it("should not allow to submit corrupted relay entry", async () => {
      // This is not a valid G1 point
      let groupSignature = "0x11134abfa2a9844a58776650e399bca3e08ab134e42595e03e3efc5a0472bcd8";

      await expectThrow(operatorContract.relayEntry(groupSignature));
  })

  it("should not allow to submit invalid relay entry", async () => {
      // Signature is a valid G1 point but it is not a signature over the
      // expected input.
      await expectThrowWithMessage(
        operatorContract.relayEntry(blsData.nextGroupSignature),
        "Invalid signature"
      );
  });

  it("should allow to submit valid relay entry", async () => {
    await operatorContract.relayEntry(blsData.groupSignature);

    assert.equal((await serviceContract.getPastEvents())[0].args['entry'].toString(),
      blsData.groupSignatureNumber.toString(), "Should emit event with generated entry"
    );
  });

  it("should allow to submit only one entry", async () => {
    await operatorContract.relayEntry(blsData.groupSignature);

    await expectThrowWithMessage(
      operatorContract.relayEntry(blsData.groupSignature),
      "Entry was submitted"
    );
  });
});