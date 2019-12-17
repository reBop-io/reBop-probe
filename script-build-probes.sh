# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

# Create dir
mkdir -p rebopagents/windows/32bits/config/ rebopagents/windows/64bits/config/ rebopagents/linux/32bits/config rebopagents/linux/64bits/config rebopagents/macosx/config/

# Copy default prod config files

cp config/config.production.yml rebopagents/windows/32bits/config/config.production.yml
cp config/config.production.yml rebopagents/windows/64bits/config/config.production.yml
cp config/config.production.yml rebopagents/linux/32bits/config/config.production.yml
cp config/config.production.yml rebopagents/linux/64bits/config/config.production.yml
cp config/config.production.yml rebopagents/macosx/config/config.production.yml

# Generate executables

echo "Building rebopagents ..."

# Windows
GOOS=windows GOARCH=386 go build -o ./rebopagents/windows/32bits/rebopagent.exe
GOOS=windows GOARCH=amd64 go build -o ./rebopagents/windows/64bits/rebopagent.exe

# Linux
GOOS=linux GOARCH=386 go build -o ./rebopagents/linux/32bits/rebopagent
GOOS=linux GOARCH=amd64 go build -o ./rebopagents/linux/64bits/rebopagent

# MacOS
GOOS=darwin GOARCH=amd64 go build -o ./rebopagents/macosx/rebopagent

# Generate Zip files

echo "Generating zip files ..."

cd rebopagents/windows/32bits/
zip ../../rebopagent-32bits-windows.zip rebopagent.exe config/config.production.yml

cd ../64bits/
zip ../../rebopagent-64bits-windows.zip rebopagent.exe config/config.production.yml

cd ../../linux/32bits/
zip ../../rebopagent-32bits-linux.zip rebopagent config/config.production.yml

cd ../64bits/
zip ../../rebopagent-64bits-linux.zip rebopagent config/config.production.yml

cd ../../macosx/
zip ../rebopagent-macosx.zip rebopagent config/config.production.yml

echo "Done"