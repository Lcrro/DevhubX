import {spawn} from 'node:child_process';
import {readFile} from 'node:fs/promises';
import path from 'node:path';
import {fileURLToPath} from 'node:url';

const project=path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const state=JSON.parse(await readFile(path.join(project,'roadmap','state.json'),'utf8'));
const port=44000+Math.floor(Math.random()*1000),base=`http://127.0.0.1:${port}`;
const child=spawn(process.execPath,[path.join(project,'roadmap','server.mjs'),'--port',String(port)],{stdio:['ignore','pipe','pipe']});
let stderr='';child.stderr.on('data',d=>stderr+=d);
async function waitForServer(){for(let i=0;i<40;i++){try{const r=await fetch(`${base}/api/state`);if(r.ok)return;}catch{}await new Promise(r=>setTimeout(r,100));}throw Error(`server did not start: ${stderr}`);}
try{
 await waitForServer();
 for(const url of ['/', '/app.js', '/style.css', '/api/state']){const r=await fetch(base+url);if(!r.ok)throw Error(`${url}: HTTP ${r.status}`);}
 const api=await (await fetch(`${base}/api/state`)).json();if(api.nodes.length!==state.nodes.length)throw Error('API node count does not match state.json');
 const referenced=new Set([...(state.documents||[]).map(d=>d.path),state.acceptance?.file,...state.nodes.flatMap(n=>n.evidence||[])]);referenced.delete('');referenced.delete(undefined);
 for(const file of referenced){const r=await fetch(`${base}/doc?path=${encodeURIComponent(file)}`);if(!r.ok)throw Error(`${file}: HTTP ${r.status}`);}
 for(const attack of ['../AGENTS.md','roadmap/server.mjs']){const r=await fetch(`${base}/doc?path=${encodeURIComponent(attack)}`);if(r.status!==404)throw Error(`unreferenced path exposed: ${attack}`);}
 console.log(`PASS: ${api.nodes.length} nodes, ${api.acceptance.length} acceptance cases, ${referenced.size} referenced documents; traversal and unreferenced files denied.`);
}finally{child.kill();}
