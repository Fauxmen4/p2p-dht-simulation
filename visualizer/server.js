const http = require('http');
const fs   = require('fs');
const path = require('path');

const topologyDir = path.resolve(__dirname, '..', 'data', 'topology');
const htmlPath    = path.join(__dirname, 'index.html');
const port        = 8888;

// import latest experiment topology from data/topology/*.json
const files = fs.readdirSync(topologyDir)
  .filter(f => f.endsWith('.json'))
  .sort();

if (files.length === 0) {
  console.error(`Failedt to fin .json files in ${topologyDir}`);
  process.exit(1);
}

const latest = files[files.length - 1];
const jsonPath = path.join(topologyDir, latest);

console.log(`Using: ${latest}`);

const topology = fs.readFileSync(jsonPath, 'utf-8');
JSON.parse(topology);

const template = fs.readFileSync(htmlPath, 'utf-8');
const html = template.replace('__TOPOLOGY__', topology);

http.createServer((req, res) => {
  res.writeHead(200, { 'Content-Type': 'text/html; charset=utf-8' });
  res.end(html);
}).listen(port, () => {
  console.log(`http://localhost:${port}`);
});