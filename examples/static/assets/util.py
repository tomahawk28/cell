import os
from os import walk

f = []
for (dirpath, dirnames, filenames) in walk("./"):
    for filename in filenames:
        if '_OFF.png' in filename:
            original_filename = dirpath + filename
            destination_filename = dirpath + '_d'.join(filename.split('_OFF'))
            print destination_filename
            os.rename(original_filename, destination_filename)
