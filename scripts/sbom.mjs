import { readFile, writeFile, mkdir } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const root = path.dirname(path.dirname(fileURLToPath(import.meta.url)));
const goMod = await readFile(path.join(root, 'go.mod'), 'utf8');
const pkg = JSON.parse(await readFile(path.join(root, 'web', 'package-lock.json'), 'utf8'));
const goBlock = goMod.match(/require \(([\s\S]*?)\)/);
const goDeps = (goBlock ? goBlock[1] : '')
  .split(/\r?\n/)
  .map((line) => line.trim())
  .filter((line) => line && !line.startsWith('//'))
  .map((line) => {
    const [name, version] = line.split(/\s+/);
    return { type: 'golang', name, version: version || '' };
  });
const npmDeps = Object.entries(pkg.packages || {})
  .filter(([name, meta]) => name && meta.version)
  .map(([name, meta]) => ({
    type: 'npm',
    name: name.replace(/^node_modules\//, '') || pkg.name,
    version: meta.version,
  }));
const sbom = {
  bomFormat: 'DevHub-SBOM',
  specVersion: '1.0',
  generatedAt: new Date().toISOString(),
  components: [...goDeps, ...npmDeps],
};
const outDir = path.join(root, 'dist');
await mkdir(outDir, { recursive: true });
await writeFile(path.join(outDir, 'sbom.json'), JSON.stringify(sbom, null, 2));
console.log(`SBOM ${sbom.components.length} components -> dist/sbom.json`);
