import launch from './launch'

suite('proxy', function() {

  let server, url;
  suiteSetup(async () => {
    [server, url] = await launch();
  });

  suiteTeardown(async () => {
    server.kill();
  });


  test('upload and proxy', async () => {

  });

})
