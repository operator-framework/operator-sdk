# SDK GPG Keys

## Current *key*pers

The keys are currently in the hands of [Joe Lanford](https://github.com/joelanford/).

## Transferring keys

In order to transfer the keys to other members of the Operator SDK admins, following the following:

1. get public GPG key from the person(s) you will transfer to
2. import their key into your keyring

```
gpg --import KEY
```

3. verify their identity, is this really their key. Video call can be useful for this.
4. sign their key

```
gpg --sign-key EMAIL-OF-USERS-KEY
```

5. export the SDK public key

```
gpg --export --armor  -o 3B2F1481D146238080B346BB052996E2A20B5C7E.pub    .asc 3B2F1481D146238080B346BB052996E2A20B5C7
```
6. export the SDK private key

```
gpg --export-secret-key --armor -o 3B2F1481D146238080B346BB052996E2A2    0B5C7E.priv.asc 3B2F1481D146238080B346BB052996E2A20B5C7E
```
7. export the SDK sub key

```
gpg --export-secret-subkeys --armor -o 3B2F1481D146238080B346BB052996    E2A20B5C7E.sub_priv.asc 3B2F1481D146238080B346BB052996E2A20B5C7E
```

8. encrypt each key for the person

```
gpg --encrypt --sign --armor -r EMAIL-OF-USERS-KEY --output 052996E2A20B5C7E.subkey.private.asc.enc 052996E2A20B5C7E.subkey.private.asc
```

9. send them the encrypted key to the user

10. user should be able to decrypt with their key.

## Updating expiration date

There will be a few people that have the keys. Those people should be able to update the expiration date. This won't have to be done until November 8, 2025.

You will want to update the date of the key:

```
gpg --edit-key (key id)
```

Once you're in the gpg console select the key , there are 2, you need to update both. I just pick a 3 year term.

```
gpg> expire
(follow prompts)
3y
gpg> save
```

You can use whatever term the team wants.

One of the resources I used: [How to change the expiration date of a GPG key](https://www.g-loaded.eu/2010/11/01/change-expiration-date-gpg-key/)

## Sending keys to keyserver

Once you have the keys updated, you should send them to a keyserver. I have a couple examples, not sure if both are needed.

```
gpg --keyserver keyserver.ubuntu.com --send-key 3B2F1481D146238080B346BB052996E2A20B5C7E
gpg --keyserver pgp.mit.edu --send-key 3B2F1481D146238080B346BB052996E2A20B5C7E
```

I *think* you only need to send it to one server, most of the commands in my shell history use `pgp.mit.edu`

## Updating secring.auto.gpg

Once you have the keys updated, you need to regenerate the keyrings that are stored in the [SDK repo](https://github.com/operator-framework/operator-sdk/tree/master/.ci/gpg).

Use the SDK key to sign and encrypt it. You need to use `--local-user` to avoid GPG from using your own key.

```
gpg --cipher-algo AES256 --output secring.auto.gpg --local-user "cncf-operator-sdk@cncf.io" --sign --symmetric 3B2F1481D146238080B346BB052996E2A20B5C7E.sub_priv.asc
```

## Updating pubring.auto

This is the public keyring. It's simply the public key. Export the public key then rename it as `pubring.auto`

```
gpg --export --armor -o 3B2F1481D146238080B346BB052996E2A20B5C7E.pub.asc 3B2F1481D146238080B346BB052996E2A20B5C7E
cp 3B2F1481D146238080B346BB052996E2A20B5C7E.pub.asc pubring.auto
```

## CI usage of keys

The GPG keys are stored in [.ci/gpg](https://github.com/operator-framework/operator-sdk/tree/master/.ci/gpg) of the Operator SDK repo.

In Github settings, there is a `GPG_PASSWORD` environment variable. It is set here in the [Environments](https://github.com/operator-framework/operator-sdk/settings/environments/172302554/edit) tab. You need to be admin.

The `GPG_PASSWORD` has been encrypted and handed to a few people. These people are the keepers of the password.

## Original process

The original keys were setup using the following article.

https://blogs.itemis.com/en/secure-your-travis-ci-releases-part-2-signature-with-openpgp