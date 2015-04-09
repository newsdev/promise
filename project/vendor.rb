#!/usr/bin/env ruby

# This script is based, in part, on the BASH vendor script in the Docker
# source code.
# https://github.com/docker/docker/blob/fd2d45d7d465fe02f159f21389b92164dbb433d3/project/vendor.sh

require 'fileutils'
require 'pathname'

$root = File.expand_path(File.join(File.dirname(__FILE__), '..'))
$vendor = File.join($root, 'vendor')

$go_path = File.expand_path(ENV['GOPATH'])
$go_path_src = File.join($go_path, 'src')

$package_name = Pathname.new($root).relative_path_from(Pathname.new($go_path_src)).to_s

# run runs a command in the root directory and raises an error if something
# goes wrong.
def run(command)
	raise "non-zero exit status: `#{command}`" if !system("cd #{$root} && #{command}")
end

# vendored_package_name provides an import package name for the given package
# that is specific to this project.
def vendored_package_name(package)
	File.join($package_name, 'vendor/src', package)
end

# vendored_package_path the absolute path to the vendored version of the given
# package.
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

	# Remove the SCM directory.
	FileUtils.rm_rf File.join(path, ".#{type}")
end

# Clear the vendor directory before doing anything.
FileUtils.rm_rf $vendor

# List the packages you actually intend to vendor.
get_package :git, 'github.com/coreos/go-etcd', '6aa2da5a7a905609c93036b9307185a04a5a84a5'
get_package :git, 'github.com/Sirupsen/logrus', 'c0f7e35ed2e48f188c37581b4b743cf7383f85c6'

# Correct the import paths in the vendored packages. We do this *after*
# fetching the packages to handle the case where one vendored package
# references another.
Dir.glob(File.join($root, 'vendor/src/**/*.go')).each do |path|

	# Check to see if the matched file is a test. We don't need to run the tests
	# of any vendored packages, and they may contain reference to packages we
	# didn't download, so we can get rid of any matching files.
	if path.match(/_test\.go$/)
		FileUtils.rm path
	else

		# Read the file and look for an import statement.
		content = File.read(path)
		if match = content.match(/(?<=import \()[^)]+/)

			# Replace references to vendored packages with their path within this
			# project.
			match[0].scan(/(?<=")[^"\n]+/).each do |package|
				if File.exists?(vendored_package_path(package))
					content.gsub!("\"#{package}\"", "\"#{vendored_package_name(package)}\"")
				end
			end

			# Write the updated content back to the file.
			File.open(path, 'w') do |file|
				file.write(content)
			end
		end
	end
end