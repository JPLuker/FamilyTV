export default function Brand() {
  return <strong className="brand">Family<b>TV</b></strong>
}

// Decorative only. The screen, dials and aerial aren't interactive controls.
export function ToyTV() {
  return <span className="toy-tv" aria-hidden="true"><i/><b/></span>
}
