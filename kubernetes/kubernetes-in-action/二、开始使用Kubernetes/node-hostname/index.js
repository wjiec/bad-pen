const os = require('os')
const http = require('http')

console.log('hostname server starting ...')

const server = http.createServer((request, response) => {
    console.log(`received request from ${request.connection.remoteAddress}`)

    response.writeHead(200)
    response.end(`You've hit <${os.hostname()}>`)
})
server.listen(8080)
