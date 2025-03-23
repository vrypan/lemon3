# lemon3

lemon3 is a file-sharing client that uses IPFS and Farcaster.

# Install

- **Binaries**: [pre-built binaries](https://github.com/vrypan/lemon3/releases)
- **Source**: Clone the repo, and run `make`. Copy the generated binaries `lemon3` to a location in your $PATH.
- **macOS/Homebrew**: `brew install vrypan/lemon3/lemon3`

## Use

Use `lemon3 --help` to see all options.

`lemon3` will start with an empty list of casts and wait for new casts containing an `enclosure+ipfs://` embed.

If you want to populate the initial view with casts from a user, use `lemon3 start <fname1> <fname2> ...`. Try `lemon3 start fc1` to see it in action (fc1 is an account I use for tests).

**Note**: `lemon3` connects to the configured Farcaster node using a gRPC streaming API. This is not open in public hubs like hoyt.farcaster.xyz. You will probably need your own hub. Configure it using
- `lemon3 config set hub.host <ip or hostname>`
- `lemon3 config set hub.ssl <true/false>`

If you leave the standard configuration with hoyt, you will not get real-time
updates -but the rest will work.

In order to upload files, you need a Farcaster appkey. The easiest way to do so is to use [CastKeys](https://www.castkeys.xyz/). Once you have your private key, run `lemon3 config set key.private <private_key>` to configure it.
