/**
 * Post-build script to clean up Next.js server files for Wails embedding.
 * This reduces the build size by removing unnecessary server-side code.
 */

const fs = require('fs');
const path = require('path');

const outDir = path.join(__dirname, '..', 'out');

// Directories to remove (server-side only, not needed for static hosting)
const dirsToRemove = [
    'server',
    'cache',
    'diagnostics',
    'types',
    'trace'
];

// Files to remove (server manifests not needed for client-side routing)
const filesToRemove = [
    'required-server-files.json',
    'next-server.js.nft.json',
    'next-minimal-server.js.nft.json'
];

function rimraf(dirPath) {
    if (fs.existsSync(dirPath)) {
        if (fs.statSync(dirPath).isDirectory()) {
            fs.readdirSync(dirPath).forEach(file => {
                rimraf(path.join(dirPath, file));
            });
            fs.rmdirSync(dirPath);
        } else {
            fs.unlinkSync(dirPath);
        }
    }
}

// Remove directories
dirsToRemove.forEach(dir => {
    const dirPath = path.join(outDir, dir);
    if (fs.existsSync(dirPath)) {
        console.log(`Removing directory: ${dir}`);
        rimraf(dirPath);
    }
});

// Remove files
filesToRemove.forEach(file => {
    const filePath = path.join(outDir, file);
    if (fs.existsSync(filePath)) {
        console.log(`Removing file: ${file}`);
        fs.unlinkSync(filePath);
    }
});

// Calculate remaining size
function getDirSize(dirPath) {
    let size = 0;
    if (fs.existsSync(dirPath)) {
        const files = fs.readdirSync(dirPath);
        files.forEach(file => {
            const filePath = path.join(dirPath, file);
            const stats = fs.statSync(filePath);
            if (stats.isDirectory()) {
                size += getDirSize(filePath);
            } else {
                size += stats.size;
            }
        });
    }
    return size;
}

const finalSize = getDirSize(outDir);
console.log(`\nBuild cleanup complete!`);
console.log(`Final output size: ${(finalSize / 1024 / 1024).toFixed(2)} MB`);
