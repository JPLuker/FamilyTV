import {useEffect,useState} from 'react'
import {errorMessage,request} from './api'

type Folder={selected:string;path:string;folders:string[]}
const label=(path:string)=>path==='.'?'Shared media':'Shared media / '+path.split('/').join(' / ')

export default function LibraryFolder({scanning,onScan}:{scanning:boolean;onScan:()=>void}){
  const [folder,setFolder]=useState<Folder>()
  const [open,setOpen]=useState(false)
  const [busy,setBusy]=useState(false)
  const [error,setError]=useState('')
  useEffect(()=>{
    let active=true
    request<Folder>('media/library-folder').then(result=>{if(active)setFolder(result)})
      .catch(e=>{if(active)setError(errorMessage(e))})
    return ()=>{active=false}
  },[])
  async function browse(path:string,save=false){
    setBusy(true);setError('')
    try{
      const result=save?await request<Folder>('media/library-folder','PUT',{path}):
        await request<Folder>('media/library-folder?path='+encodeURIComponent(path))
      setFolder(result);setOpen(!save)
      if(save)onScan()
    }catch(e){setError(errorMessage(e))}finally{setBusy(false)}
  }
  return <div>
    {folder&&<p>Library folder: <strong>{label(folder.selected)}</strong></p>}
    {!open&&<button disabled={busy} onClick={()=>browse('.')}>Choose folder</button>}
    {open&&folder&&<fieldset><legend>Choose a library folder</legend>
      <p aria-live="polite">{label(folder.path)}</p>
      {folder.path!=='.'&&<button disabled={busy} onClick={()=>browse(folder.path.split('/').slice(0,-1).join('/')||'.')}>Parent folder</button>}
      <ul>{folder.folders.map(name=><li key={name}><button disabled={busy} onClick={()=>browse(folder.path==='.'?name:folder.path+'/'+name)}>{name}</button></li>)}</ul>
      {!folder.folders.length&&<p>No subfolders.</p>}
      <button disabled={busy||scanning} onClick={()=>browse(folder.path,true)}>Use this folder and scan</button>
      <button disabled={busy} onClick={()=>{setOpen(false);setError('')}}>Cancel</button>
    </fieldset>}
    {error&&<p role="alert">{error}</p>}
  </div>
}
