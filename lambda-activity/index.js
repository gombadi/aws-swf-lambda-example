// node.js script that acts as a wrapper for Go code to run behind AWS api-gateway
var child_process = require('child_process');

exports.handler = function(event, context) {
  var proc = child_process.spawn('./gocode-amd64', [ JSON.stringify(context), JSON.stringify(event) ], { stdio: ['pipe', 'pipe', process.stderr] });

  var output = null;
  var data = null;

  proc.stdout.on('data', function(chunk) {
      if (data === null) {
          data = chunk;
      } else {
          data = Buffer.concat([data, chunk]);
      }
  });

  proc.on('close', function(code) {

    if(code > 1) {
        // completed with error so return fail so api-gateway will pick it up
      output = JSON.parse(data.toString('UTF-8'));
      return context.fail(JSON.stringify(output));
    }

    if(code == 1) {
        // for redirects just return the location
      return context.fail(data.toString());
    }

    // completed with no error so send output
    context.succeed(data.toString());
  });
}

