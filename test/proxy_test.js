import launch from './launch'
import request from 'superagent-promise';
import { S3 } from 'aws-sdk-promise';
import crypto from 'crypto';
import assert from 'assert';
import eventToPromise from 'event-to-promise';

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
    if (!e.response || !e.response.redirect) throw e;
    return e.response;
  }
  throw new Error('Did not redirect!');
}

async function get(url) {
  return await request.get(url).buffer(true).end();
}

async function getResponse(url, headers={}) {
  let req = request.get(url).redirects(0).set(headers);
  req.end();
  return await Promise.race([
    eventToPromise(req, 'error').then(() => { throw e; }),
    eventToPromise(req, 'response')
  ]);
}

function createMd5(buffer) {
  return crypto.createHash('md5')
    .update(buffer)
    .digest('hex');
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

  async function putObject(key, body) {
    // Upload the source !
    let { data } = await sourceS3.putObject({
      Body: body,
      Key: key,
      Bucket: SOURCE_BUCKET
    }).promise();
    return data;
  }

  async function upload(size) {
    // Allocate a large empty buffer for upload...
    let body = generateBuffer(size);
    let key = `proxy-test-${Date.now()}`;

    let uploadResult = await putObject(key, body);
    let md5 = createMd5(body);

    return { key, size, proxyUrl: `${url}${key}`, md5, uploadResult, body };
  }

  test('invalid (or unknown) resource in the source', async () => {
    let proxyUrl = `${url}wtfnobodyseriouslyhasthiskeyright`;
    let res = await expectRedirect(proxyUrl);
    assert(
      res.header.location.indexOf(SOURCE_BUCKET) !== -1,
      'When encountering non 200 response redirect back to source'
    );
  });

  test('second request will wait until s3 upload', async () => {
    let size = 1024 * 1024 * 1;
    let { key, md5, proxyUrl, uploadResult } = await upload(size);

    let firstResponse = await getResponse(proxyUrl);
    let secondResponse = await getResponse(proxyUrl);

    assert.equal(firstResponse.statusCode, 200);
    assert.equal(secondResponse.statusCode, 302);
    assert(
      secondResponse.headers.location.indexOf(DEST_BUCKET) !== -1,
      'Redirects to destination bucket'
    );
  });

  test('pending request timeout', async () => {
    let size = 1024 * 1024 * 50;
    let { key, md5, proxyUrl, uploadResult } = await upload(size);

    let firstResponse = await getResponse(proxyUrl);
    let secondResponse = await getResponse(proxyUrl, {
      "x-max-wait-duration": "1s"
    });

    assert.equal(firstResponse.statusCode, 200);
    assert.equal(secondResponse.statusCode, 302);
    assert(
      secondResponse.headers.location.indexOf(SOURCE_BUCKET) !== -1,
      'Redirects to source bucket'
    );
  });

  test('pass through then redirect', async () => {
    let size = 1024 * 1024 * 1;
    let { key, md5, proxyUrl, uploadResult, body } = await upload(size);
    // This is purely to validate assumptions about what aws does...
    assert.equal(`"${md5}"`, uploadResult.ETag);

    // First request should directly send content...
    let passThroughRes = await get(proxyUrl);

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
    assert.ok(
      redirectReq.headers.location.indexOf(DEST_BUCKET) !== -1,
      'Redirects to detination'
    );

    // Final sanity check to ensure thing work out of the box...
    let redirectRes = await get(proxyUrl);
    assert.equal(redirectRes.text, body.toString());
  });

})
