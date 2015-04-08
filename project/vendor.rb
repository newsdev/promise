#!/usr/bin/env ruby

packages = [
	[:git, 'github.com/coreos/go-etcd', '6aa2da5a7a905609c93036b9307185a04a5a84a5'],
	[:git, 'github.com/Sirupsen/logrus', 'c0f7e35ed2e48f188c37581b4b743cf7383f85c6'],
]

# This script is based on the BASH vendor script in the Docker source code.
# https://github.com/docker/docker/blob/fd2d45d7d465fe02f159f21389b92164dbb433d3/project/vendor.sh

require 'fileutils'
require 'pathname'

$root = File.expand_path(File.join(File.dirname(__FILE__), '..'))
$vendor = File.join($root, 'vendor')
$src = File.join($vendor, 'src')
$pkg = File.join($vendor, 'pkg')

$go_path = File.expand_path(ENV['GOPATH'])
$go_path_src = File.join($go_path, 'src')

$package_name = Pathname.new($root).relative_path_from(Pathname.new($go_path_src)).to_s

def run(command)
	raise "non-zero exit status: `#{command}`" if !system("cd #{$root} && #{command}")
end

def vendored_package_name(package)
	File.join($package_name, 'vendor/src', package)
end

def vendored_package_path(package)
	File.join($root, 'vendor/src', package)
end

def get_package(type, name, ref)

	path = vendored_package_path(name)
	FileUtils.mkdir_p path

	case type
	when :git
		run "git clone --quiet --no-checkout https://#{name} #{path}"
		run "cd #{path} && git reset --quiet --hard #{ref}"
	when :hg
		run "hg clone --quiet --updaterev #{ref} https://#{name} #{path}"
	end
end

FileUtils.rm_rf $src

packages.each do |package|
	get_package *package
end

run "GOPATH=#{$vendor} go get ./vendor/src/..."
FileUtils.rm_rf File.join($root, 'vendor/pkg')

Dir.glob(File.join($root, 'vendor/src/**/.*')).each do |path|
	FileUtils.rm_rf path
end

Dir.glob(File.join($root, 'vendor/src/**/*.go')).each do |path|

	if path.match(/_test\.go$/)
		FileUtils.rm path
	else

		content = File.read(path)
		if match = content.match(/(?<=import \()[^)]+/)

			match[0].scan(/(?<=")[^"\n]+/).each do |package|
				if File.exists?(vendored_package_path(package))
					content.gsub!("\"#{package}\"", "\"#{vendored_package_name(package)}\"")
				end
			end

			File.open(path, 'w') do |file|
				file.write(content)
			end
		end
	end
end