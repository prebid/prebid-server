require 'json'
require 'net/http'

# use: `ruby fake.rb` or `MOCK_HOST=localhost:4646 ruby fake.rb`

MOCK_ENDPOINT_JSON = <<-EOM
{
	"method": "POST",
	"path": "",
	"drain": false,
	"responses": []
}
EOM

MOCK_RESPONSE_JSON = <<-EOM
{
	"status": 200,
	"body": ""
}
EOM

def readJSON(filepath, method, path, status)
  full_filepath = File.join(File.dirname(__FILE__), filepath)
  json = File.read(full_filepath)

  response = JSON.parse(MOCK_RESPONSE_JSON)
  response['status'] = status
  response['body'] = json

  mock = JSON.parse(MOCK_ENDPOINT_JSON)
  mock['method'] = method
  mock['path'] = path
  mock['responses'] << response

  mock.to_json
end

def putEndpoint(path, json)
  mock_host = !(ENV['MOCK_HOST'].nil? || ENV['MOCK_HOST'].empty?) ? ENV['MOCK_HOST'] : 'localhost:4646'

  uri = URI.parse("http://#{mock_host}/endpoint")
  req = Net::HTTP::Put.new(uri, 'Content-Type' => 'application/json')
  req.body = json

  http = Net::HTTP.new(uri.host, uri.port)
  resp = http.request(req)

  puts "status: #{resp.code}; NetHTTPSuccess? #{resp.kind_of? Net::HTTPSuccess}; #{path}"
end


class Endpoint < Struct.new(:source, :method, :path, :status)
    def initialize(source, method, path, status=200); super end
end
endpoints = [
  Endpoint.new('../../responses/rubicon/fill_low_bid.json', 'POST', '/a/api/exchange.json'),
  Endpoint.new('../../responses/liftoff/fill_low_bid.json', 'POST', '/givemeads'),
  Endpoint.new('../../responses/moloco/fill_low_bid.json', 'POST', '/moloco_givemeads'),
  Endpoint.new('../../responses/crossinstall/fill_low_bid.json', 'POST', '/crossinstall_givemeads'),
  Endpoint.new('../../responses/taurusx/fill_low_bid.json', 'POST', '/taurusx_givemeads'),
  Endpoint.new('../../responses/unicorn/fill_low_bid.json', 'POST', '/unicorn_givemeads'),
  Endpoint.new('../../responses/pubmatic/fill_low_bid.json', 'POST', '/pubmatic_givemeads'),
  Endpoint.new('../../responses/molococloud/fill_low_bid.json', 'POST', '/molococloud_givemeads'),
  Endpoint.new('../../responses/pangle/fill_low_bid.json', 'POST', '/pangle_givemeads'),
  Endpoint.new('../../responses/dv360/fill_high_bid.json', 'POST', '/dv360_givemeads'),
]

endpoints.each do |e|
  mock_json = readJSON(e.source, e.method, e.path, e.status)
  puts mock_json unless (ENV['DEBUG'].nil? || ENV['DEBUG'].empty?)
  putEndpoint(e.path, mock_json)
end
