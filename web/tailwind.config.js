/** @type {import('tailwindcss').Config} */
export default {
	content: ['./src/**/*.{html,svelte,js,ts}'],
	theme: {
		extend: {
			fontFamily: {
				sans: ['Cairo', 'system-ui', '-apple-system', 'sans-serif']
			},
			borderRadius: {
				md: '12px',
				lg: '16px'
			},
			// Theme-aware colors — every value is a CSS variable that flips
			// between dark and light (see app.css). bodaay branding guide.
			colors: {
				bg: 'var(--c-gray-950)',
				toolbar: 'var(--c-gray-925)',
				card: 'var(--c-gray-900)',
				elevated: 'var(--c-gray-800)',
				line: 'var(--c-gray-700)',
				'line-strong': 'var(--c-gray-600)',
				ink: 'var(--c-gray-50)',
				'ink-2': 'var(--c-gray-200)',
				'ink-3': 'var(--c-gray-300)',
				'ink-4': 'var(--c-gray-400)',
				brand: 'var(--c-brand-100)',
				primary: 'var(--c-primary)',
				success: 'var(--c-success)',
				danger: 'var(--c-danger)',
				warning: 'var(--c-warning)',
				info: 'var(--c-info)'
			}
		}
	},
	plugins: []
};
