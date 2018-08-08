
// this causes geth to behave more like testrpc, only mining when there are transactions in txpool

web3.eth.filter("pending").watch(function() {
   if (eth.mining) return;
   console.log(new Date() + "-- Transactions detected, so starting mining.");
   miner.start(1);
});

web3.eth.filter('latest', function(error, result) {
   if (txpool.status.pending || !eth.mining) return;
   console.log(new Date() + "-- No pending transactions, so stopping mining.");
   miner.stop();
});

if (txpool.status.pending) {
    console.log(new Date() + "-- Pending transactions on startup, so starting mining.");
    miner.start(1);
}

console.log(new Date() + "-- Started on-demand mining. Watching txpool for pending Txs..");