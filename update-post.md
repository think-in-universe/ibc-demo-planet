## Update Post via IBC

This docs describes how to update post via IBC by adding `IbcUpdatePost` packet and callbacks.

#### 1. Add IBC Update Post Packet

```bash
ignite scaffold packet ibcUpdatePost postID title content --ack ok --module blog
```

#### 2. Add `Editor` field to `IbcUpdatePostPacketData`

To ensure only the author could update the post, in `proto/blog/packet.proto` file, add `editor` field to `IbcUpdatePostPacketData`, and rebuild the project. 

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

If the post exists and the update post request is sent by the post author, update title and content, and return `Ok` with `true` in ack.

```go
	// Set Ok to false by default, assuming errors would happen
	packetAck.Ok = false

	id, err := strconv.ParseUint(data.PostID, 10, 64)
	if err != nil {
		return packetAck, errors.New("invalid post ID")
	}

	// Check whether post exists
	post, found := k.GetPost(
		ctx,
		id,
	)
	if !found {
		return packetAck, errors.New("post ID not found")
	}

	// Permission control, only author could update post
	editor := packet.SourcePort + "-" + packet.SourceChannel + "-" + data.Editor
	if post.Creator != editor {
		return packetAck, errors.New("only the original author could update the post")
	}

	// Update title and content of the updated post
	post.Title = data.Title
	post.Content = data.Content
	k.SetPost(
		ctx,
		post,
	)

	// Set Ok to true if no errors
	packetAck.Ok = true
```

#### 4. Modify `OnAcknowledgementIbcUpdatePostPacket ` in `keeper/ibc_update_post.go`

If a `SentPost` with the same ID and creator is found, update the title.

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

However, `keeper/sent_post.go` didn't provide a way to find `SentPost` by the `PostID`, because the `ID` of `SentPost` may not be the same as `PostID`.
To make it easier to find the sent post, we also modified `keeper/sent_post.go` and `keeper/ibc_post.go`, by setting the ID of the appended `SentPost` with `PostID`.

In `keeper/sent_post.go`, use the predefined ID if set.

```diff
-	// Set the ID of the appended value
-	sentPost.Id = count
+	// If ID not set, set with count by default
+	if sentPost.Id == 0 {
+		sentPost.Id = count
+	}
```

In `keeper/ibc_post.go`, set the `PostID` as ID

```diff
+   id, err := strconv.ParseUint(packetAck.PostID, 10, 64)
+   if err != nil {
+       return errors.New("invalid post ID")
+   }
+
    k.AppendSentPost(
        ctx,
        types.SentPost{
+           Id:      id,
            Creator: data.Creator,
            PostID:  packetAck.PostID,
            Title:   data.Title,
            Chain:   packet.DestinationPort + "-" + packet.DestinationChannel,
        },
    )
```

#### 5. Modify `OnTimeoutIbcUpdatePostPacket ` in `keeper/ibc_update_post.go`

Append `TimedoutPost` record if timeout happens.

```go
	k.AppendTimedoutPost(
		ctx,
		types.TimedoutPost{
			Creator:        data.Creator,
			Title:          data.Title,
			Chain:          packet.DestinationPort + "-" + packet.DestinationChannel,
			ExistingPostID: "",
		},
	)
```

We modified `TimedoutPost` in `blog/timedout_post.proto` by adding an `existingPostID` field to differentiate new post and updated post. `existingPostID` is empty if we're creating a new post, and will be the target post ID when updating post.

```diff
message TimedoutPost {
  uint64 id = 1;
  string title = 2; 
  string chain = 3; 
  string creator = 4; 
-
+ string existingPostID = 5;
}
```

#### 6. Build `planetd`

```bash
cd cmd/planetd
go build
```

#### 7. Start `earth` and `mars` chains

```bash
# clean up chain data before running the test
rm -rf ~/.earth ~/.mars
ignite chain serve -c earth.yml
ignite chain serve -c mars.yml
```

#### 8. Start one relayer between `earth` and `mars`

```bash
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

You will find the channel ID (e.g. `channel-0`) in the output which will be used in the next step.

```bash
------
Paths
------

earth-mars:
    earth > (port: blog) (channel: channel-0)
    mars  > (port: blog) (channel: channel-0)

------
Listening and relaying packets between chains...
------
```

#### 9. Create and Update Post via IBC

Create one post with title and content

```bash
planetd tx blog send-ibc-post blog channel-0 "Hello" "Hello Mars, I'm Alice from Earth" --from alice --chain-id earth --home ~/.earth

# Via RPC query, verify the post has been created successfully
planetd q blog list-post --node tcp://localhost:26659
planetd q blog list-sent-post
```

Update the above post, with post ID, title and content

```bash
planetd tx blog send-ibc-update-post blog channel-0 0 "How are you" "Hello Mars, This is Alice from Earth" --from alice --chain-id earth --home ~/.earth

# Via RPC query, verify the post's title and content have been updated successfully
planetd q blog list-post --node tcp://localhost:26659
planetd q blog list-sent-post
```

For more details about the commands, try run `planetd tx blog`

```bash
> planetd tx blog

blog transactions subcommands

Usage:
  planetd tx blog [flags]
  planetd tx blog [command]

Available Commands:
  send-ibc-post        Send a ibcPost over IBC
  send-ibc-update-post Send a ibcUpdatePost over IBC

Flags:
  -h, --help   help for blog

Global Flags:
      --chain-id string     The network chain ID (default "planet")
      --home string         directory for config and data (default "$HOME/.planet")
      --log_format string   The logging format (json|plain) (default "plain")
      --log_level string    The logging level (trace|debug|info|warn|error|fatal|panic) (default "info")
      --trace               print out full stack trace on errors

Use "planetd tx blog [command] --help" for more information about a command.
```
