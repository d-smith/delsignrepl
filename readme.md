# delsignrepl - Demo repl for delegated signing

## Some useful demo context

Ganache address -  0x73dA1eD554De26C467d97ADE090af6d52851745E

Get balance

balance = await web3.eth.getBalance('0x73dA1eD554De26C467d97ADE090af6d52851745E')
web3.utils.fromWei(balance)

Fund from an address

web3.eth.sendTransaction({to:'0xB7Ed5c8B13176150b9b7A6fA3027Bc4818236Ed5', from:accounts[0], value: web3.utils.toWei('100')})



