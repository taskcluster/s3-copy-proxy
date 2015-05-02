import { S3 } from 'aws-sdk-promise'
import fs from 'fs';

let s3 = new S3({ region: 'us-west-2' });

async function main() {
  console.log(process.argv);
  let [, , bucket, key, path] = process.argv;
  let stream = fs.createReadStream(path);

  let res = await s3.putObject({
    Body: stream,
    Bucket: bucket,
    Key: key
  }).promise();

  console.log(res.data);
}

main().catch((err) => {
  setTimeout(() => {
    throw err;
  });
});
