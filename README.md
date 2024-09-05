Verifier XRP Indexer
====================

This is an indexer for the XRP blockchain. It is using the Verifier Indexer
Framework (https://gitlab.com/flarenetwork/fdc/verifier-indexer-framework). It
collects Blocks and Transactions from the XRP blockchain. The indexer is
configured via a configuration file - see `config.example.toml` for an example
configuration.

Installation
------------

To install the indexer, you need to have Go installed to the latest version -
currently 1.23.0.

As a first step, you need to set up your local Go environment to be able to
access private Gitlab repositories. You may have done this already by
configuring git to use the SSH URL for the repository - however this does
not seem to work with the dependencies used by this project. Instead, you
should set up a personal access token and use it to authenticate via HTTPS.
The steps to do this are as follows:

1. If git is configured to use the SSH URL, remove this configuration by
   running the following command:
```
git config --global --remove url."git@gitlab.com:"
```

2. Create a personal access token in Gitlab by going to Preferences -> Access
   Tokens -> Add New Token and creating a new token with at least the "read_api"
   scope. You can optionally choose an expiry for this token or remove it
   to keep it valid indefinitely.

3. Set up git to use the HTTPS URL for the repository by running the following:
```
echo "machine gitlab.com login <your-username> password <your-access-token>" >> ~/.netrc
```

Now you may install the indexer by running the following command:

```shell
go install gitlab.com/flarenetwork/fdc/verifier-xrp-indexer/cmd/indexer@latest
```

This will install the `indexer` command globally. Alternatively you can clone
the repository and run `go run ./cmd/indexer` to build and run the binary.

By default the indexer will look for a configuration file named `config.toml`
in the current working directory. You can specify a different configuration
file by passing `--config <filepath>` when running the indexer.
