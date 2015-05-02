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

async function expectRedirect(url) {
  // Redirect without any follows will throw..
  try {
    await request.get(url).redirects(0).end();
  } catch (e) {
    if (!e.response.redirect) throw e;
    return e.response;
  }
  throw new Error('Did not redirect!');
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
    let { data: uploadResult } = await sourceS3.putObject({
      Body: body,
      Key: key,
      Bucket: SOURCE_BUCKET
    }).promise();

    // This is purely to validate assumptions about what aws does...
    assert.equal(`"${md5}"`, uploadResult.ETag);

    // First request should directly send content...
    let proxyUrl = `${url}${key}`;
    let passThroughRes = await request.get(proxyUrl).buffer(true).end();

    // Should pass through the raw content and be of the right size...
    assert.equal(passThroughRes.status, 200);
    assert.equal(parseInt(passThroughRes.headers['content-length'], 10), size);

    // Validate that the proxy uploaded the object as well...
    let { data: head } = await destS3.headObject({
      Key: key,
      Bucket: DEST_BUCKET
    }).promise();

    assert.equal(parseInt(head.ContentLength, 10), size);
    assert.equal(head.Etag, uploadResult.Etag);

    let redirectReq = await expectRedirect(proxyUrl);
    assert.equal(redirectReq.status, 302);

    // Final sanity check to ensure thing work out of the box...
    let redirectRes = await request.get(proxyUrl).buffer(true).end();
    assert.equal(redirectRes.text, body.toString());
  });

})
