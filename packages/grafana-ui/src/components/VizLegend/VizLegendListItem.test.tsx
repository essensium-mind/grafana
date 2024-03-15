import React from 'react';
import { render } from '@testing-library/react';
import { VizLegendListItem } from './VizLegendListItem';

describe('VizLegendListItem regex functionality', () => {
    it('should correctly identify and process a Markdown link', () => {
        const testItem = {
            label: '[Markdown Label](https://example.com/)',
            yAxis: 0,
        };

        const { getByText } = render(<VizLegendListItem item={testItem} />);

        expect(getByText('Markdown Label')).toBeInTheDocument();
    });

    it('should display the label as is if not a Markdown link', () => {
        const testItem = {
            label: 'Regular Label',
            yAxis: 0,
        };

        const { getByText } = render(<VizLegendListItem item={testItem} />);

        expect(getByText('Regular Label')).toBeInTheDocument();
    })
});
