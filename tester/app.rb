require 'logger'
require 'mongo'

operations = {}
ARGV.each do |a|
    operations[a] = true
end

log = Logger.new(STDOUT)
log.level = Logger::DEBUG

address = '127.0.0.1:9999'
database = 'test'

log.info("Connecting to address=#{address} database=#{database}")
client = Mongo::Client.new([address], database: database)

log.info('Using collection people')
collection = client[:people]


if operations.key? "insert"
    doc = { name: 'Steve', hobbies: ['hiking', 'tennis', 'fly fishing'] }
    log.info("Inserting #{doc}")
    result = collection.insert_one(doc)
    puts result.n # returns 1, because one document was inserted
    log.info("Result count=#{result.n} ids=#{result.inserted_ids}")

    doc = { name: 'Steve', hobbies: ['hiking', 'tennis', 'fly fishing'] }
    log.info("Inserting #{doc}")
    result = collection.insert_one(doc)
    puts result.n # returns 1, because one document was inserted
    log.info("Result count=#{result.n} ids=#{result.inserted_ids}")
end

if operations.key? "query"
    results = collection.find({name: 'Steve'}, {limit: 10})
    results.each do |result|
        log.info("Got a result=#{result}")
    end
end

if operations.key? "drop"
    collection.drop()
end
