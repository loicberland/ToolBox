import React from 'react';

export class LocalErrorBoundary extends React.Component<{ children: React.ReactNode }, { error: string }> {
  state = { error: '' };

  static getDerivedStateFromError(error: Error) {
    return { error: error.message };
  }

  render() {
    if (this.state.error) {
      return <p className="error">{this.state.error}</p>;
    }
    return this.props.children;
  }
}
