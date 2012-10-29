Vagrant::Config.run do |config|
  config.vm.define :'rpm-i386' do |rpm32|
    rpm32.vm.box = "rpm-i386"
    rpm32.vm.box_url = "https://dl.dropbox.com/u/87191017/centos-5-i386.box"
    config.vm.customize ["setextradata", :id, "VBoxInternal2/SharedFoldersEnableSymlinksCreate/v-root", "1"]
  end

  config.vm.define :'rpm-x86_64' do |rpm64|
    rpm64.vm.box = "rpm-x86_64"
    rpm64.vm.box_url = "https://dl.dropbox.com/u/87191017/centos-5-x86_64.box"
    config.vm.customize ["setextradata", :id, "VBoxInternal2/SharedFoldersEnableSymlinksCreate/v-root", "1"]
  end

  config.vm.define :'deb-i386' do |deb32|
    deb32.vm.box = "deb-i386"
    deb32.vm.box_url = "https://dl.dropbox.com/u/87191017/debian-5-i386.box"
    config.vm.customize ["setextradata", :id, "VBoxInternal2/SharedFoldersEnableSymlinksCreate/v-root", "1"]
  end

  config.vm.define :'deb-x86_64' do |deb64|
    deb64.vm.box = "deb-x86_64"
    deb64.vm.box_url = "https://dl.dropbox.com/u/87191017/debian-5-x86_64.box"
    config.vm.customize ["setextradata", :id, "VBoxInternal2/SharedFoldersEnableSymlinksCreate/v-root", "1"]
  end
end