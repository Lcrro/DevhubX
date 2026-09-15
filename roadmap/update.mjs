import {readFile,writeFile,rename,stat} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const file=fileURLToPath(new URL('./state.json',import.meta.url));
const projectRoot=path.dirname(path.dirname(file));
const data=JSON.parse(await readFile(file,'utf8'));
const allowedStatuses=new Set(['done','active','pending','blocked','future']);
function safeRelative(value){const normalized=String(value??'').replaceAll('\\','/');return normalized&&!path.posix.isAbsolute(normalized)&&!normalized.includes(':')&&!normalized.split('/').includes('..')?normalized.replace(/^\.\//,''):null;}
function parseAcceptance(markdown){return [...markdown.matchAll(/^\|\s*\[[ xX]\]\s*\|\s*([A-Za-z][A-Za-z0-9._-]*)\b/gm)].map(m=>m[1]);}
async function exists(relative){const clean=safeRelative(relative);if(!clean)return false;try{await stat(path.join(projectRoot,...clean.split('/')));return true;}catch{return false;}}
function requireString(value,label){if(typeof value!=='string'||!value.trim())throw Error(`${label} 必须是非空字符串`);}

export async function validate(value){
 if(value.schemaVersion!==1)throw Error('仅支持 schemaVersion 1');
 for(const key of ['name','shortName','context','eyebrow','headline','versionLabel'])requireString(value.project?.[key],`project.${key}`);
 if(Number.isNaN(Date.parse(value.updatedAt)))throw Error('updatedAt 必须是有效日期时间');
 if(!Number.isInteger(value.defaultPort)||value.defaultPort<1||value.defaultPort>65535)throw Error('defaultPort 无效');
 if(!Number.isInteger(value.canvas?.width)||!Number.isInteger(value.canvas?.height)||value.canvas.width<320||value.canvas.height<320)throw Error('canvas 尺寸无效');
 if(!Array.isArray(value.stages)||!value.stages.length)throw Error('stages 不能为空');
 const stageIds=new Set();for(const s of value.stages){requireString(s.id,'stage.id');requireString(s.title,`${s.id}.title`);requireString(s.subtitle,`${s.id}.subtitle`);if(stageIds.has(s.id))throw Error(`阶段 ID 重复 ${s.id}`);stageIds.add(s.id);if(!Number.isFinite(s.y)||s.y<0||s.y>=value.canvas.height)throw Error(`${s.id}.y 无效`);}
 const acceptanceFile=safeRelative(value.acceptance?.file),acceptanceIds=[];
 if(value.acceptance?.file){if(!acceptanceFile||!(await exists(acceptanceFile)))throw Error(`验收文件不存在或越界 ${value.acceptance.file}`);acceptanceIds.push(...parseAcceptance(await readFile(path.join(projectRoot,...acceptanceFile.split('/')),'utf8')));}
 if(new Set(acceptanceIds).size!==acceptanceIds.length)throw Error('验收编号重复');
 if(!Array.isArray(value.documents))throw Error('documents 必须是数组');for(const d of value.documents){requireString(d.label,'document.label');if(!(await exists(d.path)))throw Error(`文档不存在或越界 ${d.path}`);}
 if(!Array.isArray(value.nodes)||!value.nodes.length)throw Error('nodes 不能为空');const ids=new Set(value.nodes.map(n=>n.id));if(ids.size!==value.nodes.length)throw Error('节点 ID 重复');
 const visited=new Set(),visiting=new Set(),byId=new Map(value.nodes.map(n=>[n.id,n]));
 function visit(n){if(visiting.has(n.id))throw Error('依赖存在环');if(visited.has(n.id))return;visiting.add(n.id);for(const id of n.deps){if(!byId.has(id))throw Error(`缺失前置 ${id}`);visit(byId.get(id));}visiting.delete(n.id);visited.add(n.id);}
 for(const n of value.nodes){
  requireString(n.id,'node.id');if(!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(n.id))throw Error(`节点 ID 必须是小写 slug: ${n.id}`);for(const key of ['title','subtitle','stage','description'])requireString(n[key],`${n.id}.${key}`);if(!stageIds.has(n.stage))throw Error(`${n.id}: 未知阶段 ${n.stage}`);if(!Number.isFinite(n.x)||!Number.isFinite(n.y)||n.x<0||n.y<0||n.x+226>value.canvas.width||n.y+82>value.canvas.height)throw Error(`${n.id}: 坐标超出画布`);if(!Array.isArray(n.deps)||!Array.isArray(n.checks)||!Array.isArray(n.ac)||!Array.isArray(n.evidence))throw Error(`${n.id}: 数组字段格式无效`);if(n.checks.some(c=>typeof c.text!=='string'||!c.text.trim()||typeof c.done!=='boolean')||!n.checks.length)throw Error(`${n.id}: checks 格式无效`);if(!allowedStatuses.has(n.status))throw Error(`${n.id}: 状态无效`);visit(n);
  if(n.status==='done'&&(!n.checks.every(c=>c.done)||!n.deps.every(id=>byId.get(id).status==='done')))throw Error(`${n.id}: 完成检查点和全部前置后才能标记完成`);if(n.status==='active'&&!n.deps.every(id=>byId.get(id).status==='done'))throw Error(`${n.id}: 前置未完成，不能开始`);if(n.status==='blocked'&&!String(n.blocker||'').trim())throw Error(`${n.id}: 需填写受阻原因`);if(n.status==='done'&&!n.evidence.length)throw Error(`${n.id}: 完成节点必须关联真实证据`);
  for(const evidence of n.evidence)if(!(await exists(evidence)))throw Error(`${n.id}: 证据不存在或越界 ${evidence}`);for(const ac of n.ac)if(!acceptanceIds.includes(ac))throw Error(`${n.id}: 未知验收编号 ${ac}`);
 }
 if(value.acceptance?.requireCoverage){const referenced=new Set(value.nodes.flatMap(n=>n.ac));for(const id of acceptanceIds)if(!referenced.has(id))throw Error(`验收编号未关联路线节点 ${id}`);}
 if(!Array.isArray(value.activity)||value.activity.some(a=>!a.date||!a.title||!a.text))throw Error('activity 必须包含 date、title、text');
 return {acceptanceCount:acceptanceIds.length};
}

const [id,status,note]=process.argv.slice(2);
if(id&&id!=='--check'){
 const node=data.nodes.find(n=>n.id===id);if(!node)throw Error('找不到节点');if(!status||!note)throw Error('用法: node roadmap/update.mjs <id> <status> "验证结果或变更说明"');node.status=status;if(status==='blocked')node.blocker=note;else delete node.blocker;data.updatedAt=new Date().toISOString();data.activity.unshift({date:data.updatedAt,title:`${node.title}：${status}`,text:note});await validate(data);const tmp=`${file}.tmp`;await writeFile(tmp,JSON.stringify(data,null,2)+'\n');await rename(tmp,file);console.log('状态与开发记录已更新，页面将在 2 秒内同步。');
}else{const report=await validate(data);console.log(`PASS: ${data.nodes.length} 节点，${report.acceptanceCount} 条验收，依赖无环，状态与证据引用一致。`);}
