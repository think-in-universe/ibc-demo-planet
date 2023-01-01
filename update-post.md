## Update Post via IBC


#### 1. Add IBC Update Post Packet

```bash
ignite scaffold packet ibcUpdatePost postID title content --ack ok --module blog
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

#### 3. Modify `OnRecvIbcUpdatePostPacket ` in `keeper/ibc_update_post.go`

```go
    id, err := strconv.ParseUint(data.PostID, 10, 64)
    if err != nil {
        return packetAck, errors.New("invalid post ID")
    }

    post, found := k.GetPost(
        ctx,
        id,
    )
    if !found {
        return packetAck, errors.New("post ID not found")
    } else if post.Creator != data.Editor {
        return packetAck, errors.New("only the original author could update the post")
    }

    // update title and content of the updated post
    post.Title = data.Title
    post.Content = data.Content
    k.SetPost(
        ctx,
        post,
    )

    packetAck.Ok = true
```

#### 4. Modify `OnAcknowledgementIbcUpdatePostPacket ` in `keeper/ibc_update_post.go`

```go
    if !packetAck.Ok {
        return errors.New("update post failed. No need to update sent post")
    }

    id, err := strconv.ParseUint(data.PostID, 10, 64)
    if err != nil {
        return errors.New("invalid post ID")
    }

    sentPost, found := k.GetSentPost(
        ctx,
        id,
    )
    if !found {
        return errors.New("sent post not found")
    } else if sentPost.Creator != data.Editor {
        return errors.New("only the original author could update the sent post")
    }

    // update title of sent post
    sentPost.Title = data.Title
    k.SetSentPost(
        ctx,
        sentPost,
    )
```

Please notice that to make it easier to find the sent post, we also modified `keeper/ibc_post.go`, to set the ID of the appended sent post with `PostID`

```go
    id, err := strconv.ParseUint(packetAck.PostID, 10, 64)
    if err != nil {
        return errors.New("invalid post ID")
    }

    k.AppendSentPost(
        ctx,
        types.SentPost{
            Id:      id,
            Creator: data.Creator,
            PostID:  packetAck.PostID,
            Title:   data.Title,
            Chain:   packet.DestinationPort + "-" + packet.DestinationChannel,
        },
    )
```

#### 5. Modify `OnTimeoutIbcUpdatePostPacket ` in `keeper/ibc_update_post.go`

Append timedout post record if timeout happens.

```go
	k.AppendTimedoutPost(
		ctx,
		types.TimedoutPost{
			Creator: data.Editor,
			Title:   data.Title,
			Chain:   packet.DestinationPort + "-" + packet.DestinationChannel,
			New:     false,
		},
	)
```

We modified `TimedoutPost` in `blog/timedout_post.proto` by adding a `new` field to differentiate new post and updated post.

```diff
message TimedoutPost {
  uint64 id = 1;
  string title = 2; 
  string chain = 3; 
  string creator = 4; 
-
+ bool new = 5;
}
```

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
