import { spawn } from 'mz/child_process'
import eventToPromise from 'event-to-promise';
import http from 'http';
import net from 'net';
import Debug from 'debug';

const debug = Debug('launch');

const DEFAULTS = {
  source: 'https://s3-us-west-2.amazonaws.com/test-bucket-for-any-garbage/',
  region: 'us-west-1',
  bucket: 's3-copy-proxy-tests',
  prefix: ''
}

const BUILD_DIR = `${__dirname}/..`
const TEST_TARGET = '.test-bin'


function createProc(cmd, cwd) {
  return spawn('/bin/bash', ['-c', cmd], {
    cwd: BUILD_DIR,
    stdio: 'inherit'
  });
}

export default async function(options) {
  let opts = Object.assign({}, options, DEFAULTS);

  // Run the build first...
  let buildProc = createProc(`go build -o ${TEST_TARGET}`);
  let [code] = await eventToPromise(buildProc, 'exit');
  if (code !== 0) {
    throw new Error('Failed to build go binary....');
  }

  let server = http.createServer().listen(0);
  let { port } = server.address();
  server.close();
  opts.port = port;


  let cmd = [
    `${BUILD_DIR}/${TEST_TARGET}`,
    `--port=${opts.port}`,
    `--region=${opts.region}`,
    `--source=${opts.source}`,
    `--prefix=${opts.prefix}`,
    `--bucket=${opts.bucket}`
  ].join(' ');
  let serverProc = createProc(cmd);

  // Wait until we can connect to the server...
  while (true) {
    let sock = net.connect(port);
    let ready = await Promise.race([
      eventToPromise(sock, 'error').then(() => false),
      eventToPromise(sock, 'connect').then(() => true)
    ]);
    debug('Waiting for socket ready?', ready)
    if (!ready) continue;
    debug('Socket is ready...')
    break;
  }

  return [serverProc, `http://localhost:${port}/`];
}
