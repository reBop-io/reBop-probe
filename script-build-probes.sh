# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

# Create dir
mkdir -p reBop-agents/windows/32bits/config/ reBop-agents/windows/64bits/config/ reBop-agents/linux/32bits/config reBop-agents/linux/64bits/config reBop-agents/macosx/config/

# Copy default prod config files

cp config/config.yml reBop-agents/windows/32bits/config/config.yml
cp config/config.yml reBop-agents/windows/64bits/config/config.yml
cp config/config.yml reBop-agents/linux/32bits/config/config.yml
cp config/config.yml reBop-agents/linux/64bits/config/config.yml
cp config/config.yml reBop-agents/macosx/config/config.yml

# Generate executables

echo "Building reBop-agents ..."

# Windows
GOOS=windows GOARCH=386 go build -o ./reBop-agents/windows/32bits/reBop-agent.exe
GOOS=windows GOARCH=amd64 go build -o ./reBop-agents/windows/64bits/reBop-agent.exe

# Linux
GOOS=linux GOARCH=386 go build -o ./reBop-agents/linux/32bits/reBop-agent
GOOS=linux GOARCH=amd64 go build -o ./reBop-agents/linux/64bits/reBop-agent

# MacOS
GOOS=darwin GOARCH=amd64 go build -o ./reBop-agents/macosx/reBop-agent

# Generate Zip files

echo "Generating zip files ..."

cd reBop-agents/windows/32bits/
zip ../../reBop-agent-32bits-windows.zip reBop-agent.exe config/config.yml

cd ../64bits/
zip ../../reBop-agent-64bits-windows.zip reBop-agent.exe config/config.yml

cd ../../linux/32bits/
tar cfz ../../reBop-agent-32bits-linux.tgz reBop-agent config/config.yml

cd ../64bits/
tar cfz ../../reBop-agent-64bits-linux.tgz reBop-agent config/config.yml

cd ../../macosx/
zip ../reBop-agent-macosx.zip reBop-agent config/config.yml

echo "Deleting build files"
cd ../../
rm -Rf reBop-agents/windows
rm -Rf reBop-agents/linux
rm -Rf reBop-agents/macosx
echo "Done"