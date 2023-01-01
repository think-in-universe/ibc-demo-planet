import { GeneratedType } from "@cosmjs/proto-signing";
import { MsgSendIbcUpdatePost } from "./types/planet/blog/tx";
import { MsgSendIbcPost } from "./types/planet/blog/tx";

const msgTypes: Array<[string, GeneratedType]>  = [
    ["/planet.blog.MsgSendIbcUpdatePost", MsgSendIbcUpdatePost],
    ["/planet.blog.MsgSendIbcPost", MsgSendIbcPost],
    
];

export { msgTypes }