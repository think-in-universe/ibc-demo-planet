package keeper

import (
	"errors"
	"strconv"

	"planet/x/blog/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clienttypes "github.com/cosmos/ibc-go/v5/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v5/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v5/modules/core/24-host"
)

// TransmitIbcUpdatePostPacket transmits the packet over IBC with the specified source port and source channel
func (k Keeper) TransmitIbcUpdatePostPacket(
	ctx sdk.Context,
	packetData types.IbcUpdatePostPacketData,
	sourcePort,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
) error {

	sourceChannelEnd, found := k.ChannelKeeper.GetChannel(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(channeltypes.ErrChannelNotFound, "port ID (%s) channel ID (%s)", sourcePort, sourceChannel)
	}

	destinationPort := sourceChannelEnd.GetCounterparty().GetPortID()
	destinationChannel := sourceChannelEnd.GetCounterparty().GetChannelID()

	// get the next sequence
	sequence, found := k.ChannelKeeper.GetNextSequenceSend(ctx, sourcePort, sourceChannel)
	if !found {
		return sdkerrors.Wrapf(
			channeltypes.ErrSequenceSendNotFound,
			"source port: %s, source channel: %s", sourcePort, sourceChannel,
		)
	}

	channelCap, ok := k.ScopedKeeper.GetCapability(ctx, host.ChannelCapabilityPath(sourcePort, sourceChannel))
	if !ok {
		return sdkerrors.Wrap(channeltypes.ErrChannelCapabilityNotFound, "module does not own channel capability")
	}

	packetBytes, err := packetData.GetBytes()
	if err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, "cannot marshal the packet: "+err.Error())
	}

	packet := channeltypes.NewPacket(
		packetBytes,
		sequence,
		sourcePort,
		sourceChannel,
		destinationPort,
		destinationChannel,
		timeoutHeight,
		timeoutTimestamp,
	)

	if err := k.ChannelKeeper.SendPacket(ctx, channelCap, packet); err != nil {
		return err
	}

	return nil
}

// OnRecvIbcUpdatePostPacket processes packet reception
func (k Keeper) OnRecvIbcUpdatePostPacket(ctx sdk.Context, packet channeltypes.Packet, data types.IbcUpdatePostPacketData) (packetAck types.IbcUpdatePostPacketAck, err error) {
	// validate packet data upon receiving
	if err := data.ValidateBasic(); err != nil {
		return packetAck, err
	}

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

	return packetAck, nil
}

// OnAcknowledgementIbcUpdatePostPacket responds to the the success or failure of a packet
// acknowledgement written on the receiving chain.
func (k Keeper) OnAcknowledgementIbcUpdatePostPacket(ctx sdk.Context, packet channeltypes.Packet, data types.IbcUpdatePostPacketData, ack channeltypes.Acknowledgement) error {
	switch dispatchedAck := ack.Response.(type) {
	case *channeltypes.Acknowledgement_Error:

		// TODO: failed acknowledgement logic
		_ = dispatchedAck.Error

		return nil
	case *channeltypes.Acknowledgement_Result:
		// Decode the packet acknowledgment
		var packetAck types.IbcUpdatePostPacketAck

		if err := types.ModuleCdc.UnmarshalJSON(dispatchedAck.Result, &packetAck); err != nil {
			// The counter-party module doesn't implement the correct acknowledgment format
			return errors.New("cannot unmarshal acknowledgment")
		}

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

		return nil
	default:
		// The counter-party module doesn't implement the correct acknowledgment format
		return errors.New("invalid acknowledgment format")
	}
}

// OnTimeoutIbcUpdatePostPacket responds to the case where a packet has not been transmitted because of a timeout
func (k Keeper) OnTimeoutIbcUpdatePostPacket(ctx sdk.Context, packet channeltypes.Packet, data types.IbcUpdatePostPacketData) error {

	// TODO: packet timeout logic

	return nil
}
