# delsignrepl - Demo repl for delegated signing

This project illustrates a delegated signing model. In this model, public key
encryption, in conjunction with authentication of a user via some IDP, is
used to establish the authenticity of API requests, some of which provide
access to private keys used to sign transactions for accounts/addresses owned 
by the user.

In this example, the demo code generates an RSA key pair, registering the public
key with our example backend. Wallets and addresses can be created for the user,
who must digitally sign the request to spend from an account with their private key. On the server side, the public key registered by the user is used to verify the signature, and if it checks out the parameters passed by the user are used to construct a transaction, which is then signed by the server and broadcast to the network.

The key concept is the server never has access to the user's private key, and
can build controls and policies around the use of the private key, Of course the use will have to trust the operator of the service, but any rug pull activity is
easily detectable on the public blockchain.

Note for this model something like webauthn/passkeys could be used in a key
generation and signup flow securely generate and store keys on the end user 
device, and then register the public key with the backend.

See [this page](https://webauthn.guide/) for more details on the flow.



## Some useful demo context

Ganache address -  0x73dA1eD554De26C467d97ADE090af6d52851745E

Get balance

balance = await web3.eth.getBalance('0x73dA1eD554De26C467d97ADE090af6d52851745E')
web3.utils.fromWei(balance)

Fund from an address

web3.eth.sendTransaction({to:'0xB7Ed5c8B13176150b9b7A6fA3027Bc4818236Ed5', from:accounts[0], value: web3.utils.toWei('100')})



