import terser from '@rollup/plugin-terser';
import dts from 'rollup-plugin-dts';
import esbuild from 'rollup-plugin-esbuild';

const watch = Boolean(process.env.ROLLUP_WATCH);
const minify = {
    ecma: 2020,
    compress: {
        passes: 3,
        drop_debugger: true,
    },
    format: { comments: false },
};

const browserOutput = {
    file: 'analytics.js',
    format: 'umd',
    name: 'SiraajAnalytics',
    exports: 'named',
};

const esmPackageMetadata = {
    name: 'esm-package-metadata',
    generateBundle() {
        this.emitFile({
            type: 'asset',
            fileName: 'package.json',
            source: '{"type":"module"}\n',
        });
    },
};

const productionOutputs = [
    browserOutput,
    {
        ...browserOutput,
        file: 'analytics.min.js',
        plugins: [terser({
            ...minify,
            compress: { ...minify.compress, drop_console: true },
        })],
    },
    {
        file: 'dist/analytics.esm.js',
        format: 'esm',
        plugins: [terser({ ...minify, module: true }), esmPackageMetadata],
    },
];

const builds = [
    {
        input: 'src/core/index.ts',
        output: watch ? browserOutput : productionOutputs,
        plugins: [esbuild({ target: 'es2020' })],
        treeshake: {
            preset: 'smallest',
            moduleSideEffects: false,
            propertyReadSideEffects: false,
        },
    },
];

if (!watch) {
    builds.push({
        input: 'src/core/index.ts',
        output: {
            file: 'dist/analytics.d.ts',
            format: 'esm',
        },
        plugins: [dts()],
    });
}

export default builds;
