const mack = require('@tryfabric/mack');

async function run() {
  const blocks = await mack.markdownToBlocks(process.env.INPUT.replace("\\n", "\n"));
  console.log(blocks); 
  process.exit(0);
}

run()
