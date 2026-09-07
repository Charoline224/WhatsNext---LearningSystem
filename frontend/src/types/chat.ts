import type { RetrievalResult } from './retrieval'
export interface DecisionSignal { id:string; related_node_id:string|null; status:'replanning'|'pending_plan' }
export interface ChatResult { answer:string; sources:RetrievalResult[]; decision_signal:DecisionSignal }
export interface ChatMessage { id:string; role:'user'|'assistant'; content:string; sources?:RetrievalResult[]; decisionSignal?:DecisionSignal }
