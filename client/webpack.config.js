const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = (env, argv) => {
  const isProduction = argv.mode === 'production'; // Vérifie si le mode est "production"

  return {
    entry: './src/index.tsx', // Changez l'entrée en .tsx
    // entry: './src/index.js', // Point d'entrée de l'application
    output: {
      path: path.resolve(__dirname, 'dist'), // Dossier de sortie
      filename: 'bundle.js', // Nom du fichier bundle
      clean: true, // Nettoie le dossier de sortie avant chaque build
    },
    module: {
      rules: [
        {
          test: /\.tsx?$/, // Utilisez ts-loader pour .ts et .tsx
          exclude: /node_modules/,
          use: {
            loader: 'ts-loader',
            options: {
              transpileOnly: false // Active uniquement la transpilation sans vérification des types
            }
          },
        },
        {
          test: /\.jsx?$/, // Pour tous les fichiers .js et .jsx
          exclude: /node_modules/, // Exclure le dossier node_modules
          use: {
            loader: 'babel-loader', // Utiliser babel-loader
          },
        },
      ],
    },
    resolve: {
      extensions: ['.js', '.jsx', '.ts', '.tsx'], // Extensions à résoudre
    },
    plugins: [
      new HtmlWebpackPlugin({
        template: './src/index.html', // Modèle HTML pour générer votre page
      }),
    ],
    devtool: isProduction ? 'cheap-source-map' : 'eval-source-map', // Utilise cheap-source-map en prod
    mode: isProduction ? 'production' : 'development', // Définit le mode de compilation
    devServer: {
      port: 3000, // Changer le port ici
      open: true, // Ouvrir le navigateur automatiquement
      hot: true, // Active le Hot Module Replacement
    },
  };
};
