const esbuild = require("esbuild");
const path = require("path");
const fs = require("fs");

const inputDir = "public/modules"; // Directory with your original JS files
const outputDir = "public/dist/modules"; // Output directory for minified files

// Define your entry points
const entryPoints = {
  home: path.join(inputDir, "home.js"),
  checkout: path.join(inputDir, "checkout.js"),
};

// Build options
const buildOptions = {
  entryPoints: Object.values(entryPoints),
  bundle: true,
  outdir: outputDir, // Output directory
  minify: true, // Minify in production
  format: "esm", // Use ES module format
};

// Build the project
esbuild.build(buildOptions).then(() => {
  // After the build, rename the output files to include the .min.js suffix
  Object.keys(entryPoints).forEach((key) => {
    const minifiedFile = path.join(outputDir, `${key}.js`); // Default output from esbuild
    const renamedFile = path.join(outputDir, `${key}.min.js`); // Desired output name with .min.js suffix

    // Rename the file to add the .min suffix
    fs.rename(minifiedFile, renamedFile, (err) => {
      if (err) {
        console.error(`Error renaming ${minifiedFile} to ${renamedFile}:`, err);
      } else {
        console.log(`Renamed ${minifiedFile} to ${renamedFile}`);
      }
    });
  });
}).catch(() => process.exit(1));

