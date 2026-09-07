import { apiClient, unwrap } from './client'
import type { ChatResult } from '@/types/chat'
export const chatApi={ask:(spaceId:string,message:string)=>unwrap<ChatResult>(apiClient.post(`/spaces/${spaceId}/chat`,{message,material_ids:[]}))}
