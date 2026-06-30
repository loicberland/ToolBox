import React from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { RequiredDot } from './RequiredDot';

describe('RequiredDot', () => {
  it('renders the shared required dot without visible text', () => {
    const markup = renderToStaticMarkup(<RequiredDot />);

    expect(markup).toContain('class="v10-required-dot"');
    expect(markup).toContain('aria-label="Champ obligatoire"');
    expect(markup).not.toContain('Obligatoire');
  });
});
