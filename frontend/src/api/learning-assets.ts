import { apiClient, unwrap } from './client'
import type { KnowledgeArticle, KnowledgeEdgeInput, KnowledgeNodeInput, LearningAssetJob, LearningAssets, LearningEdge, LearningNode } from '@/types/learning-asset'

export const learningAssetsApi = {
  get: (spaceId: string) => unwrap<LearningAssets>(apiClient.get(`/spaces/${spaceId}/learning-assets`)),
  generate: (spaceId: string) => unwrap<LearningAssetJob>(apiClient.post(`/spaces/${spaceId}/learning-assets/generate`)),
  generatePlan: (spaceId: string) => unwrap<LearningAssetJob>(apiClient.post(`/spaces/${spaceId}/learning-plan/generate`)),
  getJob: (spaceId: string, jobId: string) => unwrap<LearningAssetJob>(apiClient.get(`/spaces/${spaceId}/learning-assets/jobs/${jobId}`)),
  createNode: (spaceId:string,input:KnowledgeNodeInput) => unwrap<LearningNode>(apiClient.post(`/spaces/${spaceId}/knowledge-map/nodes`,input)),
  updateNode: (spaceId:string,nodeId:string,input:KnowledgeNodeInput) => unwrap<LearningNode>(apiClient.patch(`/spaces/${spaceId}/knowledge-map/nodes/${nodeId}`,input)),
  updateNodePosition: (spaceId:string,nodeId:string,position:{position_x:number;position_y:number}) => apiClient.patch(`/spaces/${spaceId}/knowledge-map/nodes/${nodeId}/position`,position),
  deleteNode: (spaceId:string,nodeId:string) => apiClient.delete(`/spaces/${spaceId}/knowledge-map/nodes/${nodeId}`),
  createEdge: (spaceId:string,input:KnowledgeEdgeInput) => unwrap<LearningEdge>(apiClient.post(`/spaces/${spaceId}/knowledge-map/edges`,input)),
  deleteEdge: (spaceId:string,edgeId:string) => apiClient.delete(`/spaces/${spaceId}/knowledge-map/edges/${edgeId}`),
  updateArticle: (spaceId:string,articleId:string,input:{title:string;body:string}) => unwrap<KnowledgeArticle>(apiClient.patch(`/spaces/${spaceId}/knowledge-handbook/articles/${articleId}`,input)),
}
