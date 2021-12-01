# https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04

# Create dir
mkdir -p reBop-probe-bin/windows/32bits/config/ \
    reBop-probe-bin/windows/64bits/config/ \
    reBop-probe-bin/linux/32bits/config \
    reBop-probe-bin/linux/64bits/config \
    reBop-probe-bin/macosx/intel/config/ \
    reBop-probe-bin/macosx/silicon/config/

# Copy default prod config files

cp config/config.yml reBop-probe-bin/windows/32bits/config/config.yml
cp config/config.yml reBop-probe-bin/windows/64bits/config/config.yml
cp config/config.yml reBop-probe-bin/linux/32bits/config/config.yml
cp config/config.yml reBop-probe-bin/linux/64bits/config/config.yml
cp config/config.yml reBop-probe-bin/macosx/intel/config/config.yml
cp config/config.yml reBop-probe-bin/macosx/silicon/config/config.yml

# Generate executables

echo "Building reBop-probe ..."

# Windows
GOOS=windows GOARCH=386 go build -o ./reBop-probe-bin/windows/32bits/reBop-probe.exe
GOOS=windows GOARCH=amd64 go build -o ./reBop-probe-bin/windows/64bits/reBop-probe.exe

# Linux
GOOS=linux GOARCH=386 go build -o ./reBop-probe-bin/linux/32bits/reBop-probe
GOOS=linux GOARCH=amd64 go build -o ./reBop-probe-bin/linux/64bits/reBop-probe

# MacOS
GOOS=darwin GOARCH=amd64 go build -o ./reBop-probe-bin/macosx/intel/reBop-probe
GOOS=darwin GOARCH=arm64 go build -o ./reBop-probe-bin/macosx/silicon/reBop-probe

# Generate Zip files

echo "Generating zip files ..."

cd reBop-probe-bin/windows/32bits/
zip ../../reBop-probe-32bits-windows.zip reBop-probe.exe config/config.yml

cd ../64bits/
zip ../../reBop-probe-64bits-windows.zip reBop-probe.exe config/config.yml

cd ../../linux/32bits/
tar cfz ../../reBop-probe-32bits-linux.tgz reBop-probe config/config.yml

cd ../64bits/
tar cfz ../../reBop-probe-64bits-linux.tgz reBop-probe config/config.yml

cd ../../macosx/intel
zip ../../reBop-probe-intel-macosx.zip reBop-probe config/config.yml

cd ../silicon
zip ../../reBop-probe-silicon-macosx.zip reBop-probe config/config.yml

echo "Deleting build files"
cd ../../
rm -Rf reBop-probe-bin/windows
rm -Rf reBop-probe-bin/linux
rm -Rf reBop-probe-bin/macosx
echo "Done"