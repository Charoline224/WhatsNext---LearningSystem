#!/usr/bin/env sh
set -eu

API_BASE="${API_BASE:-http://localhost:8080/api/v1}"
EMAIL="smoke-$(date +%s)@whatsnext.local"
PASSWORD="integration-password-123"

json_get() {
  node -e 'let b="";process.stdin.on("data",d=>b+=d).on("end",()=>{let v=JSON.parse(b);for(const k of process.argv[1].split("."))v=v[k];if(v===undefined||v===null)process.exit(2);process.stdout.write(String(v))})' "$1"
}
json_node_field() {
  node -e 'let b="";process.stdin.on("data",d=>b+=d).on("end",()=>{const v=JSON.parse(b).data.knowledge_map.nodes.find(n=>n.id===process.argv[1]);if(!v||v[process.argv[2]]===undefined)process.exit(2);process.stdout.write(String(v[process.argv[2]]))})' "$1" "$2"
}

register=$(curl -fsS -X POST "$API_BASE/auth/register" -H 'Content-Type: application/json' -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\",\"display_name\":\"链路测试\"}")
token=$(printf '%s' "$register" | json_get 'data.access_token')
auth="Authorization: Bearer $token"
space=$(curl -fsS -X POST "$API_BASE/spaces" -H "$auth" -H 'Content-Type: application/json' -d '{"name":"RAG 链路测试","mode":"exam","goal":"验证资料检索","exam_date":"2099-12-31","daily_minutes":30}')
space_id=$(printf '%s' "$space" | json_get 'data.id')
upload=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/materials" -H "$auth" -F 'file=@backend/README.md;type=text/markdown')
material_id=$(printf '%s' "$upload" | json_get 'data.material.id')

attempt=0
while [ "$attempt" -lt 60 ]; do
  status=$(curl -fsS "$API_BASE/spaces/$space_id/materials/$material_id/index-status" -H "$auth" | json_get 'data.status')
  case "$status" in
    indexed|partial) break ;;
    failed) echo "indexing failed" >&2; exit 1 ;;
  esac
  attempt=$((attempt + 1))
  sleep 1
done
[ "$attempt" -lt 60 ] || { echo "indexing timed out" >&2; exit 1; }

search=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/retrieval/search" -H "$auth" -H 'Content-Type: application/json' -d '{"query":"Qdrant 向量检索","top_k":8}')
result_count=$(printf '%s' "$search" | json_get 'data.items.length')
[ "$result_count" -gt 0 ] || { echo "retrieval returned no chunks" >&2; exit 1; }
download_url=$(curl -fsS "$API_BASE/spaces/$space_id/materials/$material_id/download-url" -H "$auth" | json_get 'data.url')
[ -n "$download_url" ] || { echo "download URL is empty" >&2; exit 1; }

asset_job=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/learning-assets/generate" -H "$auth")
asset_job_id=$(printf '%s' "$asset_job" | json_get 'data.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  asset_status=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets/jobs/$asset_job_id" -H "$auth" | json_get 'data.status')
  case "$asset_status" in
    succeeded) break ;;
    failed) echo "learning asset generation failed" >&2; exit 1 ;;
  esac
  attempt=$((attempt + 1))
  sleep 1
done
[ "$attempt" -lt 30 ] || { echo "learning asset generation timed out" >&2; exit 1; }
assets=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth")
node_count=$(printf '%s' "$assets" | json_get 'data.knowledge_map.nodes.length')
edge_count=$(printf '%s' "$assets" | json_get 'data.knowledge_map.edges.length')
article_count=$(printf '%s' "$assets" | json_get 'data.handbook.articles.length')
[ "$node_count" -gt 0 ] && [ "$edge_count" -gt 1 ] && [ "$article_count" -gt 0 ] || { echo "knowledge assets are incomplete" >&2; exit 1; }
first_node_id=$(printf '%s' "$assets" | json_get 'data.knowledge_map.nodes.0.id')
created_node=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/knowledge-map/nodes" -H "$auth" -H 'Content-Type: application/json' -d '{"name":"手动知识点","node_type":"knowledge","description":"验证地图编辑","exam_weight":0.8,"estimated_minutes":20,"position_x":420,"position_y":180}')
created_node_id=$(printf '%s' "$created_node" | json_get 'data.id')
curl -fsS -X PATCH "$API_BASE/spaces/$space_id/knowledge-map/nodes/$created_node_id" -H "$auth" -H 'Content-Type: application/json' -d '{"name":"已修改知识点","node_type":"skill","description":"验证节点修改","exam_weight":0.9,"estimated_minutes":25,"position_x":440,"position_y":200}' >/dev/null
curl -fsS -X PATCH "$API_BASE/spaces/$space_id/knowledge-map/nodes/$created_node_id/position" -H "$auth" -H 'Content-Type: application/json' -d '{"position_x":460,"position_y":220}' >/dev/null
created_edge=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/knowledge-map/edges" -H "$auth" -H 'Content-Type: application/json' -d "{\"from_node_id\":\"$first_node_id\",\"to_node_id\":\"$created_node_id\",\"relation_type\":\"related\"}")
created_edge_id=$(printf '%s' "$created_edge" | json_get 'data.id')
curl -fsS -X DELETE "$API_BASE/spaces/$space_id/knowledge-map/edges/$created_edge_id" -H "$auth" >/dev/null
curl -fsS -X DELETE "$API_BASE/spaces/$space_id/knowledge-map/nodes/$created_node_id" -H "$auth" >/dev/null
article_id=$(printf '%s' "$assets" | json_get 'data.handbook.articles.0.id')
curl -fsS -X PATCH "$API_BASE/spaces/$space_id/knowledge-handbook/articles/$article_id" -H "$auth" -H 'Content-Type: application/json' -d '{"title":"人工保护章节","body":"这是不能被 AI 重新生成覆盖的人工正文。"}' >/dev/null
auto_plan_id=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth" | json_get 'data.plan_job.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  auto_plan_status=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets/jobs/$auto_plan_id" -H "$auth" | json_get 'data.status')
  case "$auto_plan_status" in succeeded) break ;; failed) echo "handbook-triggered plan failed" >&2; exit 1 ;; esac
  attempt=$((attempt + 1)); sleep 1
done
merge_job=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/learning-assets/generate" -H "$auth")
merge_job_id=$(printf '%s' "$merge_job" | json_get 'data.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  merge_status=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets/jobs/$merge_job_id" -H "$auth" | json_get 'data.status')
  case "$merge_status" in succeeded) break ;; failed) echo "protected knowledge merge failed" >&2; exit 1 ;; esac
  attempt=$((attempt + 1)); sleep 1
done
merged_assets=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth")
protected_body=$(printf '%s' "$merged_assets" | json_get 'data.handbook.articles.0.body')
[ "$protected_body" = "这是不能被 AI 重新生成覆盖的人工正文。" ] || { echo "manual handbook content was overwritten" >&2; exit 1; }

plan_job=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/learning-plan/generate" -H "$auth")
plan_job_id=$(printf '%s' "$plan_job" | json_get 'data.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  plan_status=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets/jobs/$plan_job_id" -H "$auth" | json_get 'data.status')
  case "$plan_status" in
    succeeded) break ;;
    failed) echo "learning plan generation failed" >&2; exit 1 ;;
  esac
  attempt=$((attempt + 1))
  sleep 1
done
[ "$attempt" -lt 30 ] || { echo "learning plan generation timed out" >&2; exit 1; }
assets=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth")
task_count=$(printf '%s' "$assets" | json_get 'data.today_plan.tasks.length')
[ "$task_count" -gt 0 ] || { echo "learning plan is incomplete" >&2; exit 1; }
exam_upload=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/materials" -H "$auth" -F 'material_kind=past_exam' -F 'file=@scripts/fixtures/sample-exam.txt;type=text/plain')
exam_job_id=$(printf '%s' "$exam_upload" | json_get 'data.job.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  exam_job_status=$(curl -fsS "$API_BASE/spaces/$space_id/jobs/$exam_job_id" -H "$auth" | json_get 'data.status')
  case "$exam_job_status" in succeeded) break ;; failed) echo "exam analysis failed" >&2; exit 1 ;; esac
  attempt=$((attempt + 1)); sleep 1
done
exam_analysis=$(curl -fsS "$API_BASE/spaces/$space_id/exam-analysis" -H "$auth")
question_count=$(printf '%s' "$exam_analysis" | json_get 'data.questions.length')
pattern_count=$(printf '%s' "$exam_analysis" | json_get 'data.patterns.length')
pattern_knowledge=$(printf '%s' "$exam_analysis" | json_get 'data.patterns.0.tested_knowledge')
pattern_node_count=$(printf '%s' "$exam_analysis" | json_get 'data.patterns.0.related_nodes.length')
pattern_node_id=$(printf '%s' "$exam_analysis" | json_get 'data.patterns.0.related_nodes.0.node_id')
[ "$question_count" -eq 4 ] && [ "$pattern_count" -ge 3 ] && [ -n "$pattern_knowledge" ] && [ "$pattern_node_count" -ge 1 ] || { echo "exam was not split into semantic patterns linked to knowledge nodes" >&2; exit 1; }
question_id=$(printf '%s' "$exam_analysis" | json_get 'data.questions.0.id')
feedback=$(curl -fsS -X PUT "$API_BASE/spaces/$space_id/exam-questions/$question_id/feedback" -H "$auth" -H 'Content-Type: application/json' -d '{"is_correct":false,"note":"滑动窗口概念混淆"}')
feedback_status=$(printf '%s' "$feedback" | json_get 'data.decision_status')
[ "$feedback_status" = "replanning" ] || { echo "exam feedback did not trigger replanning" >&2; exit 1; }
mastery_assets=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth")
mastery_status=$(printf '%s' "$mastery_assets" | json_node_field "$pattern_node_id" 'mastery_status')
mastery_evidence=$(printf '%s' "$mastery_assets" | json_node_field "$pattern_node_id" 'evidence_count')
[ "$mastery_status" = "weak" ] && [ "$mastery_evidence" -ge 1 ] || { echo "wrong answer did not update node mastery" >&2; exit 1; }
feedback_plan_id=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth" | json_get 'data.plan_job.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  feedback_plan_status=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets/jobs/$feedback_plan_id" -H "$auth" | json_get 'data.status')
  case "$feedback_plan_status" in succeeded) break ;; failed) echo "exam-feedback plan failed" >&2; exit 1 ;; esac
  attempt=$((attempt + 1)); sleep 1
done
chat=$(curl -fsS -X POST "$API_BASE/spaces/$space_id/chat" -H "$auth" -H 'Content-Type: application/json' -d '{"message":"Qdrant 向量检索是什么？"}')
chat_answer=$(printf '%s' "$chat" | json_get 'data.answer')
chat_source_count=$(printf '%s' "$chat" | json_get 'data.sources.length')
decision_status=$(printf '%s' "$chat" | json_get 'data.decision_signal.status')
[ -n "$chat_answer" ] && [ "$chat_source_count" -gt 0 ] && [ "$decision_status" = "replanning" ] || { echo "space chat did not trigger a grounded replan" >&2; exit 1; }
replan_job_id=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth" | json_get 'data.plan_job.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  replan_status=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets/jobs/$replan_job_id" -H "$auth" | json_get 'data.status')
  case "$replan_status" in
    succeeded) break ;;
    failed) echo "chat-triggered replanning failed" >&2; exit 1 ;;
  esac
  attempt=$((attempt + 1))
  sleep 1
done
[ "$attempt" -lt 30 ] || { echo "chat-triggered replanning timed out" >&2; exit 1; }
plan_reason=$(curl -fsS "$API_BASE/spaces/$space_id/learning-assets" -H "$auth" | json_get 'data.today_plan.plan.generation_reason')
case "$plan_reason" in *"学习决策信号"*) ;; *) echo "plan did not record learning signals" >&2; exit 1 ;; esac
curl -fsS -X DELETE "$API_BASE/spaces/$space_id" -H "$auth" >/dev/null

exam_only_space=$(curl -fsS -X POST "$API_BASE/spaces" -H "$auth" -H 'Content-Type: application/json' -d '{"name":"仅真题规划测试","mode":"exam","goal":"从真题错题决策计划","exam_date":"2099-12-31","daily_minutes":30}')
exam_only_space_id=$(printf '%s' "$exam_only_space" | json_get 'data.id')
exam_only_upload=$(curl -fsS -X POST "$API_BASE/spaces/$exam_only_space_id/materials" -H "$auth" -F 'material_kind=past_exam' -F 'file=@scripts/fixtures/sample-exam.txt;type=text/plain')
exam_only_job_id=$(printf '%s' "$exam_only_upload" | json_get 'data.job.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  exam_only_status=$(curl -fsS "$API_BASE/spaces/$exam_only_space_id/jobs/$exam_only_job_id" -H "$auth" | json_get 'data.status')
  case "$exam_only_status" in succeeded) break ;; failed) echo "exam-only analysis failed" >&2; exit 1 ;; esac
  attempt=$((attempt + 1)); sleep 1
done
exam_only_analysis=$(curl -fsS "$API_BASE/spaces/$exam_only_space_id/exam-analysis" -H "$auth")
exam_only_question_id=$(printf '%s' "$exam_only_analysis" | json_get 'data.questions.0.id')
exam_only_feedback=$(curl -fsS -X PUT "$API_BASE/spaces/$exam_only_space_id/exam-questions/$exam_only_question_id/feedback" -H "$auth" -H 'Content-Type: application/json' -d '{"is_correct":false}')
exam_only_decision=$(printf '%s' "$exam_only_feedback" | json_get 'data.decision_status')
[ "$exam_only_decision" = "replanning" ] || { echo "exam-only feedback could not plan" >&2; exit 1; }
exam_only_plan_id=$(curl -fsS "$API_BASE/spaces/$exam_only_space_id/learning-assets" -H "$auth" | json_get 'data.plan_job.id')
attempt=0
while [ "$attempt" -lt 30 ]; do
  exam_only_plan_status=$(curl -fsS "$API_BASE/spaces/$exam_only_space_id/learning-assets/jobs/$exam_only_plan_id" -H "$auth" | json_get 'data.status')
  case "$exam_only_plan_status" in succeeded) break ;; failed) echo "exam-only plan failed" >&2; exit 1 ;; esac
  attempt=$((attempt + 1)); sleep 1
done
exam_only_task_count=$(curl -fsS "$API_BASE/spaces/$exam_only_space_id/learning-assets" -H "$auth" | json_get 'data.today_plan.tasks.length')
[ "$exam_only_task_count" -gt 0 ] || { echo "exam-only plan has no tasks" >&2; exit 1; }
curl -fsS -X DELETE "$API_BASE/spaces/$exam_only_space_id" -H "$auth" >/dev/null
echo "RAG smoke test passed: material=$material_id status=$status nodes=$node_count edges=$edge_count articles=$article_count tasks=$task_count exam_questions=$question_count patterns=$pattern_count feedback=$feedback_status chat_sources=$chat_source_count decision=$decision_status"
