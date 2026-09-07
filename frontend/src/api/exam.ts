import { apiClient,unwrap } from './client'
import type { ExamAnalysis,ExamFeedbackResult } from '@/types/exam'
export const examApi={get:(spaceId:string)=>unwrap<ExamAnalysis>(apiClient.get(`/spaces/${spaceId}/exam-analysis`)),feedback:(spaceId:string,questionId:string,isCorrect:boolean)=>unwrap<ExamFeedbackResult>(apiClient.put(`/spaces/${spaceId}/exam-questions/${questionId}/feedback`,{is_correct:isCorrect,note:''}))}
