# lemon3

lemon3 is a file-sharing system that uses IPFS and Farcaster.

Check out [lemon3: Farcaster+IPFS = Decentralized RSS enclosures and file sharing](https://blog.vrypan.net/2025/03/23/lemon3-farcaster-ipfs-decentralized-file-sharing/) about the name,
the idea and what's next.

**UPDATE, 2025-06-15**: `lemon3` was initially released in March 2025. It was a proof-of-concept
verison, that worked, but I had made various development decisions that eventually blocked me. So,
3 months later, I decided to implement the same idea from scratch. The repo history has been deleted
and this is a totally new codebase.

# Install

- **Binaries**: [pre-built binaries](https://github.com/vrypan/lemon3/releases)
- **Source**: Clone the repo, and run `make`. Copy the generated binaries `lemon3` to a location in your $PATH.
- **macOS/Homebrew**: `brew install vrypan/lemon3/lemon3`

# Use

Use `lemon3 --help` to see all options.

Keep in mind that you will need access to a [snapchain node](https://github.com/farcasterxyz/snapchain)
and an IPFS node (for most users, the [IPFS Desktop app](https://docs.ipfs.tech/install/ipfs-desktop/) is
the simplest choice for this).

Future versions will try to bundle these components with lemon3.


# Example

```
lemon3 upload cnApr14.mp3 \
--title="Answers for Shel Israel about the origins of blogging and RSS" \
--description="An episode of Morning Coffee Notes posted on April 14, 2005

\"Music and Red Couch answers about blogging, RSS, and who knows whatnot.\"

Dave answers questions sent in by Shel Israel. He discusses his history with blogging and the origins of RSS. He started blogging in the late 1990s as a way to communicate with a community he had created, and saw blogging as a way to bypass the traditional media that he felt did not accurately represent the software he was developing. Dave was an early pioneer of RSS, working with Netscape to create a standard format, and he describes the process of collaborating with them to establish RSS as the dominant syndication format. He reflects on the challenges of establishing standards and the importance of being open to adopting others' ideas rather than stubbornly pushing one's own.

Source: https://mcn.archive.podnews.net/cnapr14.html" \
--artwork=artwork2.png \
--cast="Dave Winer has influenced the way I see the Internet, the Web, and software development in general. Listening to this episode from 2005, reminded me why."
```

Output:

```
[^] Uploading cnApr14.mp3: 100.0%  (cid=QmPAnBDf2CHxNJeVKq1nepDXrTkn7MwRnVWjeofhtrhzES)
[+] QmPAnBDf2CHxNJeVKq1nepDXrTkn7MwRnVWjeofhtrhzES pinned.
[^] Uploading artwork2.png: 100.0%  (cid=QmPSfzSKnRDnTj2FJoNRPLC3iKaK1BbTNRkF6QLbPKd9zL)
[+] QmPSfzSKnRDnTj2FJoNRPLC3iKaK1BbTNRkF6QLbPKd9zL pinned.
[^] Metadata cid=bafyreigssoonfshbr6pxzbqjgbt7hmluy7fziuzu3wynysnxyalihxt4be
[+] bafyreigssoonfshbr6pxzbqjgbt7hmluy7fziuzu3wynysnxyalihxt4be pinned.
[^] Cast posted: @vrypan.eth/0xcd3141a47b98685c292b55c44f932e221753e51b
```

And here is the result: https://farcaster.xyz/vrypan.eth/0xcd3141a47b98685c292b55c44f932e221753e51b

You can also check this one for video embeds: https://farcaster.xyz/fc1/0xbbcba55feeef8b522843b1d73c8f9dec3a2f4f7a
