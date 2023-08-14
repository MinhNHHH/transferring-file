# transferring file

<p align="center">
  <img src="./demo.gif" alt="animated" width="800" height="400"/>
</p>

`transfer` is a tool that allows any two computers to simply and securely transfer file. `transfer` is the only CLI file-transfer tool that does all of the following:

- allows **any two computers** to transfer data (using a peer to peer)
- enables easy **cross-plaftform** transfer (Window, Linux, Mac)
- local server or port-forwarding **not needed**

The motivation behind transfer is to provide a safe and fast way to send file.
In order to achieve that, transfer uses a combination of WebSocket and WebRTC:
- WebSocket - is used only for signaling - which is a process to establish WebRTC connection
- [WebRTC](https://webrtc.org) - the primary connection to stream your terminal to other clients

# Getting started

## Install
### Git clone
1. Go to https://github.com/MinhNHHH/transferring-file clone project
2. On terminal run command `go build cmd/transferring-file/transfer.go`
3. Move it to `/usr/local/bin` folder so that you could use `transfer` anywhere : `mv transfer /usr/local/bin`

## Usage

1. Go to `www` and run command `npm run start`
2. To start a new session, just run `transfer send {{file_name}}`.
3. Transfer-file will echo out a connection url you can use to connect via:
  - Receive file with command: `transfer {{connection_url}}`

### Note
There are chances where a direct peer-to-peer connection can't be established, so I included a TURN server that I created using [CoTURN](https://github.com/coturn/coturn).

If relay to the TURN server is something you don't want, you can:
- Disable the usage of turn server (with `-no-turn` flag)
- Creates your own TURN server connect to it by changing in [cfg/transfer.go](internal/cfg/transferring-file.go)

## Upcoming
- [x] Send file
- [x] Connect to transfer session via `transfer` itself
- [ ] Send text
- [ ] Send folder
- [ ] Install via brew/apt

## Similar projects
- [https://github.com/](https://github.com/schollz/croc)https://github.com/schollz/croc
