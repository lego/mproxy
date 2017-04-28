require 'logger'
require 'mongo'

log = Logger.new(STDOUT)
log.level = Logger::DEBUG

address = '127.0.0.1:27018'
database = 'test'

log.info("Connecting to address=#{address} database=#{database}")
client = Mongo::Client.new([address], database: database)

log.info('Using collection people')
collection = client[:people]
doc = { name: 'Steve', hobbies: ['hiking', 'tennis', 'fly fishing'] }
log.info("Inserting #{doc}")
result = collection.insert_one(doc)
puts result.n # returns 1, because one document was inserted
log.info("Result count=#{result.n} ids=#{result.inserted_ids}")
