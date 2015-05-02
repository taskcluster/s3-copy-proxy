import launch from './launch'
import request from 'superagent-promise';
import { S3 } from 'aws-sdk-promise';
import crypto from 'crypto';
import assert from 'assert';

const SOURCE_BUCKET = 'test-bucket-for-any-garbage';
const DEST_BUCKET = 's3-copy-proxy-tests';

// very sketchy method of generating somewhat readable random bytes...
function generateBuffer(bytes) {
  let buf = new Buffer(bytes);
  for (let i = 0; i < bytes; i++) {
    // Get vaugely readable ascii
    let ascii = Math.floor(47 + (Math.random() * 80) + 1);
    buf.writeInt8(ascii, i);
  }
  return buf;
}

suite('proxy', function() {
  // TODO: Refactor S3 tests to be generic enough to run outside of our
  //       mozilla-taskcluster account.
  let sourceS3 = new S3({ region: 'us-west-2' });
  let destS3 = new S3({ region: 'us-east-1' });

  let server, url;
  suiteSetup(async () => {
    [server, url] = await launch();
  });

  suiteTeardown(async () => {
    server.kill();
  });

  test('upload and proxy', async () => {
    // Allocate a large empty buffer for upload...
    let time = Date.now();
    let size = 1024 * 1024 * 1;
    let body = generateBuffer(size);
    let key = `proxy-test-${Date.now()}`;

    let md5 = crypto.createHash('md5')
      .update(body)
      .digest('hex');

    // Upload the source !
    let uploadResult = await sourceS3.putObject({
      Body: body,
      Key: key,
      Bucket: SOURCE_BUCKET
    }).promise();

    // This is purely to validate assumptions about what aws does...
    assert.equal(`"${md5}"`, uploadResult.data.ETag);

    // First request should directly send content...
    let proxyUrl = `${url}${key}`;
    let passThroughRes = await request.get(proxyUrl).buffer(true).end();

    // Should pass through the raw content and be of the right size...
    assert.equal(passThroughRes.status, 200);
    assert.equal(parseInt(passThroughRes.headers['content-length'], 10), size);
  });

})
