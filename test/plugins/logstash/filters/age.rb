require "logstash/filters/base"
require "logstash/namespace"

class LogStash::Filters::Age < LogStash::Filters::Base
  config_name "age"
  plugin_status "experimental"

  def register; end

  def filter(event)
    return unless filter?(event)
    event["age"] = Time.now - event.ruby_timestamp
    filter_matched(event)
  end
end

