const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = {
  entry: './src/index.js', // Point d'entrée de l'application
  output: {
    path: path.resolve(__dirname, 'dist'), // Dossier de sortie
    filename: 'bundle.js',  // Nom du fichier bundle
    clean: true, // Nettoie le dossier de sortie avant chaque build
  },
  module: {
    rules: [
      {
        test: /\.jsx?$/, // Pour tous les fichiers .js et .jsx
        exclude: /node_modules/, // Exclure le dossier node_modules
        use: {
          loader: 'babel-loader' // Utiliser babel-loader
        },
      },
    ],
  },
  resolve: {
    extensions: ['.js', '.jsx'], // Extensions à résoudre
  },
  plugins: [
    new HtmlWebpackPlugin({
      template: './src/index.html', // Modèle HTML pour générer votre page
    }),
  ],
  devtool: 'source-map', // Facilite le débogage du code
  mode: 'development', // Mode de développement
  devServer: {
    port: 4000, // Changer le port ici
    open: true, // Ouvrir le navigateur automatiquement
  },
};
