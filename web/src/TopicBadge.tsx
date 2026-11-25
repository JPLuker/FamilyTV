// Subjects come from the parent's review. Keep unfamiliar subjects browsable too.
const subjects: [RegExp,string,string][] = [
  [/rocket|launch|spacecraft/i,'🚀','space'],
  [/space|moon|planet|astronom|orbit|star|solar/i,'🪐','space'],
  [/animal|bird|fish|bug|insect|wildlife|ocean/i,'🐾','animal'],
  [/plant|forest|tree|nature|flower|garden|earth/i,'🌱','nature'],
  [/art|draw|paint|craft|music/i,'🎨','art'],
  [/science|weather|experiment|physics/i,'🔬','science'],
  [/number|math|shape/i,'🔢','math'],
]
export function TopicBadge({subject}:{subject:string}){
  const match=subjects.find(([pattern])=>pattern.test(subject))
  return <span aria-hidden="true" className={'topic-art '+(match?.[2]||'other')}>{match?.[1]||'✦'}</span>
}
