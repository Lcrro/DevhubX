import http from 'node:http';
import {readFile} from 'node:fs/promises';
import {fileURLToPath} from 'node:url';
import path from 'node:path';

const root=path.dirname(fileURLToPath(import.meta.url));
const project=path.dirname(root);
const statePath=path.join(root,'state.json');

function safeRelative(value){
 const normalized=String(value??'').replaceAll('\\','/');
 if(!normalized||path.posix.isAbsolute(normalized)||normalized.includes(':')||normalized.split('/').includes('..'))return null;
 return normalized.replace(/^\.\//,'');
}
function acceptanceRows(markdown){
 return markdown.split(/\r?\n/).flatMap(line=>{
  if(!/^\|\s*\[[ xX]\]\s*\|/.test(line))return [];
  const cells=line.split('|').slice(1,-1).map(s=>s.trim());
  const match=cells[1]?.match(/^([A-Za-z][A-Za-z0-9._-]*)\s*(?:\/\s*(.*))?$/);
  if(!match)return [];
  return [{id:match[1],requirement:match[2]||match[1],action:cells[2]||'',criteria:cells[3]||'',level:cells[4]||'',passed:/x/i.test(cells[0])}];
 });
}
async function load(){
 const data=JSON.parse(await readFile(statePath,'utf8'));
 const source=safeRelative(data.acceptance?.file);
 let acceptance=[];
 if(source)acceptance=acceptanceRows(await readFile(path.join(project,...source.split('/')),'utf8'));
 return {...data,acceptance,acceptanceSource:source||''};
}
async function allowedFiles(data){
 const values=[...(data.documents||[]).map(d=>d.path),data.acceptance?.file,...data.nodes.flatMap(n=>n.evidence||[])];
 return new Set(values.map(safeRelative).filter(Boolean));
}
const arg=process.argv.indexOf('--port'),argPort=arg>=0?Number(process.argv[arg+1]):NaN;
const initial=await load();
const port=Number.isInteger(argPort)&&argPort>0&&argPort<65536?argPort:Number(process.env.ROADMAP_PORT||initial.defaultPort||4318);

http.createServer(async(req,res)=>{
 try{
  if(req.method!=='GET'){res.writeHead(405);return res.end();}
  const url=new URL(req.url,'http://localhost');
  if(url.pathname==='/favicon.ico'){res.writeHead(204);return res.end();}
  if(url.pathname==='/api/state'){
   const data=await load();res.writeHead(200,{'Content-Type':'application/json; charset=utf-8','Cache-Control':'no-store'});return res.end(JSON.stringify(data));
  }
  let file;
  if(url.pathname==='/doc'){
   const requested=safeRelative(url.searchParams.get('path')),data=await load(),allowed=await allowedFiles(data);
   if(!requested||!allowed.has(requested)){res.writeHead(404);return res.end();}
   file=path.join(project,...requested.split('/'));
  }else{
   const files={'/':'index.html','/app.js':'app.js','/style.css':'style.css','/state.json':'state.json'};
   if(!files[url.pathname]){res.writeHead(404);return res.end();}file=path.join(root,files[url.pathname]);
  }
  const mime={'.html':'text/html','.css':'text/css','.js':'text/javascript','.mjs':'text/javascript','.json':'application/json','.md':'text/plain','.txt':'text/plain'};
  res.writeHead(200,{'Content-Type':(mime[path.extname(file)]||'application/octet-stream')+'; charset=utf-8','Cache-Control':'no-store','X-Content-Type-Options':'nosniff'});res.end(await readFile(file));
 }catch(error){res.writeHead(500,{'Content-Type':'application/json; charset=utf-8'});res.end(JSON.stringify({error:'路线图数据暂时不可用。',detail:String(error.message||error)}));}
}).listen(port,'127.0.0.1',()=>console.log(`Roadmap: http://127.0.0.1:${port}`));
