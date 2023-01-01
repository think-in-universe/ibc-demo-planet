## Update Post via IBC


#### 1. Add IBC Update Post Packet

```bash
ignite scaffold packet ibcUpdatePost postID title content --ack postID --module blog
```

#### 2. Add `Editor` field to `IbcUpdatePostPacketData`

In `proto/blog/packet.proto` file, add `editor` field to `IbcUpdatePostPacketData`, rebuild the project. 

```bash
ignite chain build
```

In `x/blog/keeper/msg_server_ibc_update_post.go`, set `Editor` with `msg.Creator` when sending the packet.

```go
	// Construct the packet
	var packet types.IbcUpdatePostPacketData

	packet.PostID = msg.PostID
	packet.Title = msg.Title
	packet.Content = msg.Content
	packet.Editor = msg.Creator
```

#### 3. Modify `OnRecvIbcUpdatePostPacket ` in keeper


#### 4. Modify `OnAcknowledgementIbcUpdatePostPacket ` in keeper


#### 5. Modify `OnTimeoutIbcUpdatePostPacket ` in keeper


#### 6. Launch two chains

```
ignite chain serve -c earth.yml

ignite chain serve -c mars.yml
```

#### 7. Start relayer

```
rm -rf ~/.ignite/relayer

ignite relayer configure -a \
  --source-rpc "http://0.0.0.0:26657" \
  --source-faucet "http://0.0.0.0:4500" \
  --source-port "blog" \
  --source-version "blog-1" \
  --source-gasprice "0.0000025stake" \
  --source-prefix "cosmos" \
  --source-gaslimit 300000 \
  --target-rpc "http://0.0.0.0:26659" \
  --target-faucet "http://0.0.0.0:4501" \
  --target-port "blog" \
  --target-version "blog-1" \
  --target-gasprice "0.0000025stake" \
  --target-prefix "cosmos" \
  --target-gaslimit 300000

ignite relayer connect
```

#### 8. Update Post from Earth

```
planetd tx blog send-ibc-update-post blog channel-0 0 "Hello" "Hello Mars, I'm Alice from Earth" --from alice --chain-id earth --home ~/.earth
```

#### 9. Verify result via RPC query

```
planetd q blog list-post --node tcp://localhost:26659

planetd q blog list-sent-post
```
