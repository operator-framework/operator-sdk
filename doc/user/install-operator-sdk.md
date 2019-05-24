# Install the Operator SDK CLI

## Install from GitHub release

### Download the release binary

```sh
# Set the release version variable
RELEASE_VERSION=v0.8.0
# Linux
$ curl -OJL https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
# macOS
$ curl -OJL https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin
```

#### Verify the downloaded release binary

```sh
# Linux
curl -OJL https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
# macOS
curl -OJL https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
```

To verify a release binary using the provided asc files, place the binary and corresponding asc file into the same directory and use the corresponding command:

```sh
# Linux
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc
# macOS
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
```

If you do not have the maintainers public key on your machine, you will get an error message similar to this:

```sh
$ gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin.asc
$ gpg: assuming signed data in 'operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin'
$ gpg: Signature made Fri Apr  5 20:03:22 2019 CEST
$ gpg:                using RSA key <KEY_ID>
$gpg: Can't check signature: No public key
```

To download the key, use the following command, replacing `$KEY_ID` with the RSA key string provided in the output of the previous command:

```sh
$ gpg --recv-key "$KEY_ID"
```

You'll need to specify a key server if one hasn't been configured. For example:

```sh
$ gpg --keyserver keyserver.ubuntu.com --recv-key "$KEY_ID"
```

Now you should be able to verify the binary.


### Install the release binary in your PATH

```
# Linux
$ chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && sudo cp operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk && rm operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu
# macOS
$ chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin && sudo cp operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin /usr/local/bin/operator-sdk && rm operator-sdk-${RELEASE_VERSION}-x86_64-apple-darwin
```

## Install from Homebrew

Alternatively, if you are using [Homebrew][homebrew_tool], you can install the SDK CLI tool with the following command:

```sh
$ brew install operator-sdk
```

## Compile and install from master

```sh
$ mkdir -p $GOPATH/src/github.com/operator-framework
$ cd $GOPATH/src/github.com/operator-framework
$ git clone https://github.com/operator-framework/operator-sdk
$ cd operator-sdk
$ git checkout master
$ make dep
$ make install
```

[homebrew_tool]:https://brew.sh/
