@echo off
echo Committing and pushing lead group/community update...

git add -A
git commit -m "Add group and community tracking to leads - Update lead columns when adding to groups/communities"

echo.
echo Pushing to GitHub...
git push origin main

echo.
echo Done! Lead group/community tracking feature has been pushed to GitHub.
pause
